package db

import (
	"database/sql"
	"hwproj/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"fmt"
	_"log"
	"golang.org/x/crypto/bcrypt"
	"errors"
)

type Config struct {
	ConnectString string
}

func InitDb(cfg Config) (*pgDb, error) {
	if dbConn, err := sqlx.Connect("postgres", cfg.ConnectString); err != nil {
		return nil, err
	} else {
		p := &pgDb{dbConn: dbConn}
		if err := p.dbConn.Ping(); err != nil {
			return nil, err
		}
		if err := p.createTablesIfNotExist(); err != nil {
			return nil, err
		}
		if err := p.prepareSqlStatements(); err != nil {
			return nil, err
		}
		return p, nil
	}
}

type pgDb struct {
	dbConn *sqlx.DB

	sqlSelectPeople *sqlx.Stmt
	sqlInsertPerson *sql.Stmt
	sqlSelectPerson *sql.Stmt
	sqlExistsPerson *sql.Stmt
}

func (p *pgDb) createTablesIfNotExist() error {
	create_sql := `

       CREATE TABLE IF NOT EXISTS users (
        userid SERIAL UNIQUE,
        firstname TEXT NOT NULL CHECK (firstname SIMILAR TO '[(А-яа-яё\-)]{2,}'),
        surname TEXT NOT NULL CHECK (surname SIMILAR TO '[(А-яа-яё\-)]{2,}'),
        email TEXT NOT NULL CHECK (email LIKE '%@%') UNIQUE,
        password TEXT NOT NULL,
        type TEXT CHECK (type in ('student', 'teacher', 'admin')));

    `
	if rows, err := p.dbConn.Query(create_sql); err != nil {
		return err
	} else {
		rows.Close()
	}
	return nil
}

func (p *pgDb) prepareSqlStatements() (err error) {

	if p.sqlSelectPeople, err = p.dbConn.Preparex(
		"SELECT userid, firstname, surname, email, password FROM users",
	); err != nil {
		return err
	}

	if p.sqlInsertPerson, err = p.dbConn.Prepare(
		"INSERT INTO users(firstname, surname, email, password, type) VALUES($1,$2,$3,$4, 'student');",
	); err != nil {
		return err
	}

	if p.sqlSelectPerson, err = p.dbConn.Prepare(
		"SELECT userid, firstname, surname FROM users WHERE userid = $1",
	); err != nil {
		return err
	}

	if p.sqlExistsPerson, err = p.dbConn.Prepare(
		"SELECT userid, password FROM users WHERE email = $1",
	); err != nil {
		return err
	}

	return nil
}

func (p *pgDb) SelectPeople() ([]*model.Person, error) {
	users := make([]*model.Person, 0)
	if err := p.sqlSelectPeople.Select(&users); err != nil {
		return nil, err
	}
	return users, nil
}

func (p *pgDb) Insert(person model.Person) (error) {
	if _, err := p.sqlInsertPerson.Exec(person.Firstname, person.Surname, person.Email, person.Password); err != nil {
		return err
	}
	return nil
}

func (p *pgDb) Exists(data model.EntryData) (int, error) {
	row := p.sqlExistsPerson.QueryRow(data.Email)
	var id int
	var hashFromDB string
	switch err := row.Scan(&id, &hashFromDB); err {
	case sql.ErrNoRows:
		fmt.Println("There is no users with this email!")
		return -1, errors.New("There is no users with this email!")
	case nil:
		if errInPass := bcrypt.CompareHashAndPassword([]byte(hashFromDB), []byte(data.Pass)); errInPass != nil {
			fmt.Println("Wrong password!")
			return -2, errors.New("Wrong password!")
		}
		fmt.Println(id, "Password was correct")
	default:
		panic(err)
	}
	return id, nil
}