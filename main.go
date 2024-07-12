package migrator

import (
	"database/sql"
	"os"
	"strings"
)

// UTIL FUNCTIONS
func pop(l *[]string) string {
    f := len(*l)
    rv := (*l)[f-1]
    *l = (*l)[:f-1]
    return rv
}

func kv(m map[string][]any) (string, []any) {
    for k, v := range m {
        return k, v
    }
    return "", make([]any, 0)
}

// TODO: Err can be done last.
type Err struct {
    message string
}

func (e *Err) Error() string {
    return e.message
}

type Migrator struct {
    Conn *sql.DB
    QueryMap map[string]map[string][]any
}

// @NewMigrator 
// returns an instance of the Migrator struct with a provided connection reference.
func NewMigrator(conn *sql.DB) (Migrator, error) {
    m := Migrator{
        Conn: conn,
        QueryMap: make(map[string]map[string][]any),
    }
    m.AddQueriesToMap("util_queries")

    // Attempt to create the base table if needed.
    err := m.InitMigrationTable()
    if err != nil {
        return Migrator{}, err 
    }

    return m, nil
}

func (m *Migrator) AddArgsToQuery(queryName string, args []any) (error) {
    val, ok := m.QueryMap[queryName]
    if !ok {
        return &Err{ message: "queryName was not valid."} 
    }

    k, _ := kv(val)
    m.QueryMap[queryName][k] = args

    return nil
}

// TODO: Eventually this should be able to ingest a custom table via the QueryMap
func (m *Migrator) InitMigrationTable() (error) {
    query, _ := kv(m.QueryMap["util_queries/initial_schema"]) 
    _, err := m.Conn.Query(query)
    if err != nil {
        return err
    }
    return nil
}

/* 
@Migrator.AddQueriesToMap
Given a dirPath edits the Migrator.QueryMap to contain a map of [queryName]=>[queryFileContents]
It is a non-nested map where nested dirs are applied to the queryName. (also .sql file extensions are stripped from the naming convention)
example:

q/
--queryOne.sql
--/nested
----queryTwo.sql
----/nestedAgain
------queryThree.sql

would output:
queryOne=>[queryOneContents]: [...args], nested/queryTwo=>[queryTwoContents]: [...args], nested/nestedAgain/queryThree=>[queryThreeContents]: [...args]
*/
// TODO:  should skip files that are not .sql, also may need to clean the dirPath (ie; "./q" seemed to have issues comapred to just "q")
// TODO: should also run a validator on file names! migrations are order based operands and each one needs a numbered tag ending-> anything_001.sql, canbe_002.sql, here_003.sql etc.
func (m *Migrator) AddQueriesToMap(dirPath string) (error) {
    dirs := []string{ dirPath }
    for len(dirs) > 0 {
        dir := pop(&dirs)
        
        queryFiles, err := os.ReadDir(dir)
        if err != nil {
            return err
        }

        for _, file := range queryFiles {
            if (file.IsDir()) {
                dirs = append(dirs, (dir + "/" + file.Name()))
                continue
            }

            data, err := os.ReadFile(dir + "/" + file.Name())
            if err != nil {
                return err
            }
            cleanName := strings.Split(file.Name(), ".sql")
            d := make(map[string][]any)
            d[string(data)] = make([]any, 0)
            m.QueryMap[dir + "/" + cleanName[0]] = d
        }
    }
    return nil
}
