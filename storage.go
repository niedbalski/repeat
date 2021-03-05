package main

import (
	"fmt"
	"github.com/go-orm/gorm"
	_ "github.com/go-orm/gorm/dialects/sqlite"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	log "github.com/sirupsen/logrus"
	"path"
)

type DBStorage struct {
	*gorm.DB
	Tables map[string]bool
}

func NewDBStorage(dBPath string) (*DBStorage, error) {
	db, err := gorm.Open("sqlite3", path.Join(dBPath, "collections.db"))
	if err != nil {
		return nil, err
	}
	return &DBStorage{DB: db, Tables: make(map[string]bool)}, nil
}

func (db *DBStorage) CreateTable(tableName string, fields []MapValueField) {
	log.Debugf("Creating table: %s on database", tableName)
	if _, ok := db.Tables[tableName]; ok {
		log.Debugf("Table %s already exists, skipping", tableName)
		return
	}

	instance := dynamicstruct.ExtendStruct(gorm.Model{})

	for _, field := range fields {
		if field.Type == "float" {
			instance.AddField(Capitalize(field.Name), 0.0, "")
		} else if field.Type == "int" {
			instance.AddField(Capitalize(field.Name), 0, "")
		} else {
			//defaults to string type
			instance.AddField(Capitalize(field.Name), field.Type, "")
		}
	}

	newInst := instance.Build().New()

	table := db.Table(tableName)
	if table.HasTable(newInst) {
		table.AutoMigrate(newInst)
	} else {
		table.CreateTable(newInst)
	}

	db.Tables[tableName] = true
}

type InsertRecord struct {
	TableName  string
	FieldNames []string
	Values     []string
}

func (db *DBStorage) CreateRecord(task *SchedulerTask, tableName string, fields []MapValueField, values []string) error {

	var insertIntoDB = func(table string, fields []MapValueField, values []string) error {
		var fieldNames, formattedValues []string

		if table == "" || fields == nil || values == nil {
			return fmt.Errorf("Skipping data insertion, nil values passed to insertIntoDb")
		}

		log.Debugf("creating new record entry on table: %s", table)

		fieldNames = append(fieldNames, "created_at")
		for _, field := range fields {
			fieldNames = append(fieldNames, field.Name)
		}

		formattedValues = append(formattedValues, "datetime('now')")
		for _, field := range fields {
			formattedValues = append(formattedValues, field.Format(values))
		}

		if !task.Scheduler.Stopped {
			*task.DBOpsQueue <- &InsertRecord{FieldNames: fieldNames, Values: formattedValues, TableName: table}
		}
		return nil
	}

	var isIndexOnValues = func(values []string) error {
		for _, field := range fields {
			if field.Index > len(values) {
				return fmt.Errorf(
					"Not found value that matches field: %s with idx: %d in returned values (length: %d)",
					field.Name, field.Index, len(values))
			}
		}
		return nil
	}

	RemoveEmptyFromSlice(&values)
	if len(values) <= 0 {
		return fmt.Errorf("Empty set of values returned, skipping")
	}
	if err := isIndexOnValues(values); err != nil {
		return err
	}
	return insertIntoDB(tableName, fields, values)
}
