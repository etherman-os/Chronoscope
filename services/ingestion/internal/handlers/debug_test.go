package handlers

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDebugMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE sessions SET status = 'completed', completed_at = NOW\(\) WHERE id = \$1`).
		WithArgs("sess-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := db.Exec(`UPDATE sessions SET status = 'completed', completed_at = NOW() WHERE id = $1`, "sess-123")
	fmt.Printf("res=%v err=%v\n", res, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		fmt.Println("expectations not met:", err)
	}
}
