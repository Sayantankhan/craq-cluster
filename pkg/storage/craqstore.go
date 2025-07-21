package storage

import (
	"context"
	"fmt"
	"strings"

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

	return &CraqStore{pool: pool}, nil
}

func (store *CraqStore) Put(seq uint64, fileName, folder, path string) error {
	return crdbpgx.ExecuteTx(context.Background(), store.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `
			INSERT INTO chunk_metadata (folder, file_name, seq, state, path)
			VALUES ($1, $2, $3, 'dirty', $4)
			ON CONFLICT (folder, file_name) DO UPDATE
			SET seq = EXCLUDED.seq,
			    state = 'dirty',
			    path = EXCLUDED.path
			WHERE chunk_metadata.seq < EXCLUDED.seq
		`, folder, fileName, seq, path)

		return err
	})
}

func (store *CraqStore) MarkClean(folder, fileName string, seq uint64) error {
	return crdbpgx.ExecuteTx(context.Background(), store.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		tag, err := tx.Exec(context.Background(), `
			UPDATE chunk_metadata
			SET state = 'clean'
			WHERE folder = $1 AND file_name = $2 AND seq = $3
		`, folder, fileName, seq)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("no matching chunk to mark clean")
		}
		return nil
	})
}

func (store *CraqStore) GetLatest(folder, fileName string) (Chunk, bool) {
	var seq uint64
	var stateStr, path string

	err := store.pool.QueryRow(context.Background(),
		`SELECT seq, state, path 
		 FROM chunk_metadata 
		 WHERE folder = $1 AND file_name = $2`,
		folder, fileName).Scan(&seq, &stateStr, &path)

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
		Folder:   folder,
		FileName: fileName,
		Seq:      seq,
		State:    stateVersion,
		Path:     path,
	}, true
}

func (store *CraqStore) ListFilesInFolder(folder string) ([]string, error) {
	rows, err := store.pool.Query(context.Background(),
		`SELECT DISTINCT folder, file_name FROM chunk_metadata WHERE folder LIKE $1 || '%'`, folder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make(map[string]struct{})

	for rows.Next() {
		var fullFolder, fileName string
		if err := rows.Scan(&fullFolder, &fileName); err != nil {
			return nil, err
		}

		trimmed := strings.TrimPrefix(fullFolder, folder)
		if trimmed == "" {
			// Direct file in the folder
			entries[fileName] = struct{}{}
		} else {
			// Handle subdir case like `/craq/docker` → returns `docker`
			trimmed = strings.TrimPrefix(trimmed, "/")
			parts := strings.Split(trimmed, "/")
			if len(parts) > 0 && parts[0] != "" {
				entries[parts[0]+"/"] = struct{}{}
			}
		}
	}

	var result []string
	for name := range entries {
		result = append(result, name)
	}
	return result, nil
}
