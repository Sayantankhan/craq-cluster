package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
)

type CraqStore struct {
	pool *pgxpool.Pool
}

func NewCraqStore(ctx context.Context, dsn string) (*CraqStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS chunk_metadata (
			chunk_id TEXT PRIMARY KEY,
			seq BIGINT NOT NULL,
			state TEXT NOT NULL,
			file_name TEXT NOT NULL,
			path TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return &CraqStore{pool: pool}, nil
}

func (store *CraqStore) Put(chunkId string, seq uint64, fileName, path string) error {
	return crdbpgx.ExecuteTx(context.Background(), store.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `
			INSERT INTO chunk_metadata (chunk_id, seq, state, file_name, path)
			VALUES ($1, $2, 'dirty', $3, $4)
			ON CONFLICT (chunk_id) DO UPDATE
			SET seq = EXCLUDED.seq,
			    state = 'dirty',
			    file_name = EXCLUDED.file_name,
			    path = EXCLUDED.path
			WHERE chunk_metadata.seq < EXCLUDED.seq
		`, chunkId, seq, fileName, path)

		return err
	})
}

func (store *CraqStore) MarkClean(chunkId string, seq uint64) error {
	return crdbpgx.ExecuteTx(context.Background(), store.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		tag, err := tx.Exec(context.Background(), `
			UPDATE chunk_metadata
			SET state = 'clean'
			WHERE chunk_id = $1 AND seq = $2
		`, chunkId, seq)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("no matching chunk to mark clean")
		}
		return nil
	})
}

func (store *CraqStore) GetLatest(chunkId string) (Chunk, bool) {
	var seq uint64
	var stateStr string
	var fileName, path string

	err := store.pool.QueryRow(context.Background(),
		`SELECT seq, state, file_name, path FROM chunk_metadata WHERE chunk_id = $1`,
		chunkId).Scan(&seq, &stateStr, &fileName, &path)

	if err != nil {
		if err == pgx.ErrNoRows {
			return Chunk{}, false
		}
		panic(err)
	}

	var stateVersion VersionState
	if stateStr == "clean" {
		stateVersion = Clean
	} else {
		stateVersion = Dirty
	}

	return Chunk{
		ChunkID:  chunkId,
		FileName: fileName,
		Seq:      seq,
		State:    stateVersion,
		Path:     path,
	}, true
}
