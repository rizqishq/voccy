package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func createBoard(ctx context.Context, board Board) error {
	_, err := pool.Exec(ctx,
		`insert into boards (id, org_id, name, slug, description, is_active, settings, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		board.ID, board.OrgID, board.Name, board.Slug, board.Description, board.IsActive, board.Settings, board.CreatedAt, board.UpdatedAt,
	)
	return err
}

func getBoards(ctx context.Context) ([]Board, error) {
	rows, err := pool.Query(ctx,
		`select b.id, b.org_id, o.name, b.name, b.slug, b.description, b.is_active, b.settings, b.created_at, b.updated_at
		 from boards b
		 join organizations o on o.id = b.org_id`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	boards := []Board{}
	for rows.Next() {
		var b Board
		err := rows.Scan(&b.ID, &b.OrgID, &b.OrgName, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return boards, nil
}

func getBoardByID(ctx context.Context, id uuid.UUID) (*Board, error) {
	var b Board
	row := pool.QueryRow(ctx,
		`select b.id, b.org_id, o.name, b.name, b.slug, b.description, b.is_active, b.settings, b.created_at, b.updated_at
		 from boards b
		 join organizations o on o.id = b.org_id
		 where b.id=$1`,
		id,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.OrgName, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	return &b, nil
}

func getBoardBySlug(ctx context.Context, slug string) (*Board, error) {
	var b Board
	row := pool.QueryRow(ctx,
		`select b.id, b.org_id, o.name, b.name, b.slug, b.description, b.is_active, b.settings, b.created_at, b.updated_at
		 from boards b
		 join organizations o on o.id = b.org_id
		 where b.slug=$1`,
		slug,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.OrgName, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	return &b, nil
}

func getBoardByIDForOrg(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*Board, error) {
	var b Board
	row := pool.QueryRow(ctx,
		`select id, org_id, name, slug, description, is_active, settings, created_at, updated_at from boards where id=$1 and org_id=$2`,
		id, orgID,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	return &b, nil
}

func updateBoardForOrg(ctx context.Context, id uuid.UUID, orgID uuid.UUID, name, description string, settings BoardSettings) (*Board, error) {
	slug := generateSlug(name)

	var b Board
	row := pool.QueryRow(ctx,
		`update boards set name=$1, slug=$2, description=$3, settings=$4, updated_at=now()
		 where id=$5 and org_id=$6
		 returning id, org_id, name, slug, description, is_active, settings, created_at, updated_at`,
		name, slug, description, settings, id, orgID,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	return &b, nil
}

func deleteBoardForOrg(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	result, err := pool.Exec(ctx, `delete from boards where id=$1 and org_id=$2`, id, orgID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func createFeedback(ctx context.Context, feedback Feedback) error {
	_, err := pool.Exec(ctx,
		`insert into feedbacks (id, board_id, title, body, author_name, author_email, status, vote_count, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, $7, 0, $8, $9)`,
		feedback.ID, feedback.BoardID, feedback.Title, feedback.Body, feedback.AuthorName, feedback.AuthorEmail, feedback.Status, feedback.CreatedAt, feedback.UpdatedAt,
	)
	return err
}

func getFeedbacksByBoardID(ctx context.Context, boardID uuid.UUID) ([]Feedback, error) {
	rows, err := pool.Query(ctx,
		`select id, board_id, title, body, author_name, author_email, status, vote_count, created_at, updated_at
		 from feedbacks where board_id=$1 order by created_at desc`,
		boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feedbacks := []Feedback{}
	for rows.Next() {
		var f Feedback
		err := rows.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.VoteCount, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		feedbacks = append(feedbacks, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return feedbacks, nil
}

func getFeedbackByID(ctx context.Context, id uuid.UUID, boardID uuid.UUID) (*Feedback, error) {
	var f Feedback
	row := pool.QueryRow(ctx,
		`select id, board_id, title, body, author_name, author_email, status, vote_count, created_at, updated_at
		 from feedbacks where id=$1 and board_id=$2`,
		id, boardID,
	)

	if err := row.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.VoteCount, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}

	return &f, nil
}

func updateFeedbackStatusForOrg(ctx context.Context, id uuid.UUID, boardID uuid.UUID, orgID uuid.UUID, status string) (*Feedback, error) {
	var f Feedback
	row := pool.QueryRow(ctx,
		`update feedbacks f set status=$1, updated_at=now()
		 from boards b
		 where f.id=$2 and f.board_id=b.id and b.org_id=$3 and f.board_id=$4
		 returning f.id, f.board_id, f.title, f.body, f.author_name, f.author_email, f.status, f.vote_count, f.created_at, f.updated_at`,
		status, id, orgID, boardID,
	)

	if err := row.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.VoteCount, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}

	return &f, nil
}

func deleteFeedbackForOrg(ctx context.Context, id uuid.UUID, boardID uuid.UUID, orgID uuid.UUID) error {
	result, err := pool.Exec(ctx,
		`delete from feedbacks f
		 using boards b
		 where f.id=$1 and f.board_id=b.id and b.org_id=$2 and f.board_id=$3`,
		id, orgID, boardID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func toggleVote(ctx context.Context, feedbackID uuid.UUID, fingerprint string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var existingID uuid.UUID
	err = tx.QueryRow(ctx,
		`select id from votes where feedback_id=$1 and fingerprint=$2`,
		feedbackID, fingerprint,
	).Scan(&existingID)

	if err == nil {
		_, err = tx.Exec(ctx, `delete from votes where id=$1`, existingID)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `update feedbacks set vote_count = vote_count - 1 where id=$1`, feedbackID)
		if err != nil {
			return false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}

	if err != pgx.ErrNoRows {
		return false, err
	}

	_, err = tx.Exec(ctx,
		`insert into votes (feedback_id, fingerprint) values ($1, $2)`,
		feedbackID, fingerprint,
	)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, `update feedbacks set vote_count = vote_count + 1 where id=$1`, feedbackID)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

func createOrg(ctx context.Context, org Organization) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO organizations (id, name, email, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		org.ID, org.Name, org.Email, org.PasswordHash, org.CreatedAt, org.UpdatedAt,
	)
	return err
}

func getOrgByEmail(ctx context.Context, email string) (*Organization, error) {
	var org Organization
	row := pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at, updated_at FROM organizations WHERE email=$1`,
		email,
	)

	if err := row.Scan(&org.ID, &org.Name, &org.Email, &org.PasswordHash, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return nil, err
	}

	return &org, nil
}

func getOrgByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var org Organization
	row := pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at, updated_at FROM organizations WHERE id=$1`,
		id,
	)

	if err := row.Scan(&org.ID, &org.Name, &org.Email, &org.PasswordHash, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return nil, err
	}

	return &org, nil
}

func updateOrg(ctx context.Context, id uuid.UUID, name, email string) (*Organization, error) {
	var org Organization
	row := pool.QueryRow(ctx,
		`UPDATE organizations SET name=$1, email=$2, updated_at=now()
		 WHERE id=$3
		 RETURNING id, name, email, password_hash, created_at, updated_at`,
		name, email, id,
	)

	if err := row.Scan(&org.ID, &org.Name, &org.Email, &org.PasswordHash, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return nil, err
	}

	return &org, nil
}
