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
		`select id, org_id, name, slug, description, is_active, settings, created_at, updated_at from boards`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	boards := []Board{}
	for rows.Next() {
		var b Board
		err := rows.Scan(&b.ID, &b.OrgID, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt)
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
		`select id, org_id, name, slug, description, is_active, settings, created_at, updated_at from boards where id=$1`,
		id,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	return &b, nil
}

func updateBoard(ctx context.Context, id uuid.UUID, name, description string, settings BoardSettings) (*Board, error) {
	slug := generateSlug(name)

	var b Board
	row := pool.QueryRow(ctx,
		`update boards set name=$1, slug=$2, description=$3, settings=$4, updated_at=now()
		 where id=$5
		 returning id, org_id, name, slug, description, is_active, settings, created_at, updated_at`,
		name, slug, description, settings, id,
	)

	if err := row.Scan(&b.ID, &b.OrgID, &b.Name, &b.Slug, &b.Description, &b.IsActive, &b.Settings, &b.CreatedAt, &b.UpdatedAt); err != nil {
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

func deleteBoard(ctx context.Context, id uuid.UUID) error {
	result, err := pool.Exec(ctx, `delete from boards where id=$1`, id)
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
		`insert into feedbacks (id, board_id, title, body, author_name, author_email, status, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		feedback.ID, feedback.BoardID, feedback.Title, feedback.Body, feedback.AuthorName, feedback.AuthorEmail, feedback.Status, feedback.CreatedAt, feedback.UpdatedAt,
	)
	return err
}

func getFeedbacksByBoardID(ctx context.Context, boardID uuid.UUID) ([]Feedback, error) {
	rows, err := pool.Query(ctx,
		`select id, board_id, title, body, author_name, author_email, status, created_at, updated_at
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
		err := rows.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.CreatedAt, &f.UpdatedAt)
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

func getFeedbackByID(ctx context.Context, id uuid.UUID) (*Feedback, error) {
	var f Feedback
	row := pool.QueryRow(ctx,
		`select id, board_id, title, body, author_name, author_email, status, created_at, updated_at
		 from feedbacks where id=$1`,
		id,
	)

	if err := row.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}

	return &f, nil
}

func updateFeedbackStatus(ctx context.Context, id uuid.UUID, status string) (*Feedback, error) {
	var f Feedback
	row := pool.QueryRow(ctx,
		`update feedbacks set status=$1, updated_at=now() where id=$2
		 returning id, board_id, title, body, author_name, author_email, status, created_at, updated_at`,
		status, id,
	)

	if err := row.Scan(&f.ID, &f.BoardID, &f.Title, &f.Body, &f.AuthorName, &f.AuthorEmail, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}

	return &f, nil
}

func deleteFeedback(ctx context.Context, id uuid.UUID) error {
	result, err := pool.Exec(ctx, `delete from feedbacks where id=$1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
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
