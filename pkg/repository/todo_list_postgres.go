package repository

import (
	todo "ToDoApp"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"strings"
)

type TodoListPostgres struct {
	db *sqlx.DB
}

func NewTodoListPostgres(db *sqlx.DB) *TodoListPostgres {
	return &TodoListPostgres{db: db}
}

func (r *TodoListPostgres) Create(userId int, list todo.TodoList) (int, error) {
	tx, err := r.db.Begin() // команда для создания транзакции в БД (транзакция это послед-ть команд SQL, которые
	if err != nil {         // должны быть выполнены полностью, либо не выполнены вообще)
		return 0, err
	}

	// createListQuery - содержит текст запроса к БД, если транзакция не прошла то Rollback - происходит откат
	var id int
	createListQuery := fmt.Sprintf("INSERT INTO %s (title, description) VALUES ($1, $2) RETURNING id", todoListsTable)
	row := tx.QueryRow(createListQuery, list.Title, list.Description)
	if err := row.Scan(&id); err != nil {
		tx.Rollback()
		return 0, err
	}

	//делаем вставку в таблицу userLists в которой свяжем id пользователя и id списка
	createUsersListQuery := fmt.Sprintf("INSERT INTO %s (user_id, lists_id) VALUES ($1, $2)", usersListsTable)
	_, err = tx.Exec(createUsersListQuery, userId, id) // для простого выполнения запроса без чтения возвражаемой инф-ии
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit() // по выполнению транзакции выполняем commit
	// которая выполнит все изменения транзацкции в базе
}

func (r *TodoListPostgres) GetAll(userId int) ([]todo.TodoList, error) {
	var lists []todo.TodoList
	query := fmt.Sprintf("SELECT tl.id, tl.title, tl.description FROM %s tl INNER JOIN %s ul on tl.id = ul.lists_id WHERE ul.user_id = $1",
		todoListsTable, usersListsTable)
	err := r.db.Select(&lists, query, userId)

	return lists, err
}

func (r *TodoListPostgres) GetById(userId, listId int) (todo.TodoList, error) {
	var list todo.TodoList
	query := fmt.Sprintf(`SELECT tl.id, tl.title, tl.description FROM %s tl 
		INNER JOIN %s ul on tl.id = ul.lists_id WHERE ul.user_id = $1 AND ul.lists_id = $2`,
		todoListsTable, usersListsTable)
	err := r.db.Get(&list, query, userId, listId)

	return list, err
}

func (r *TodoListPostgres) Delete(userId, listId int) error {
	query := fmt.Sprintf("DELETE FROM %s tl USING %s ul WHERE tl.id = ul.lists_id AND ul.user_id=$1 AND ul.lists_id=$2",
		todoListsTable, usersListsTable)
	_, err := r.db.Exec(query, userId, listId)

	return err
}

func (r *TodoListPostgres) Update(userId, listId int, input todo.UpdateListInput) error {
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if input.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *input.Title)
		argId++
	}

	if input.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *input.Description)
		argId++
	}

	setQuery := strings.Join(setValues, ", ")
	query := fmt.Sprintf("UPDATE %s tl SET %s FROM %s ul WHERE tl.id = ul.lists_id AND ul.lists_id=$%d AND ul.user_id=$%d",
		todoListsTable, setQuery, usersListsTable, argId, argId+1)
	args = append(args, listId, userId)

	logrus.Debugf("updateQuery: %s", query)
	logrus.Debugf("args: %s", args)

	_, err := r.db.Exec(query, args...)
	return err
}
