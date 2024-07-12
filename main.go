package migrator

import (
	"database/sql"
	"os"
	"strings"
    "strconv"
    "reflect"
    "sort"
)

// UTIL FUNCTIONS
func pop(l *[]string) string {
    f := len(*l)
    rv := (*l)[f-1]
    *l = (*l)[:f-1]
    return rv
}

func (m *Migrator) Query(queryName string) ([]any, error) {
    var result []any
    val, ok := m.QueryMap[queryName]
    if !ok {
        return result, &Err{ message: "Query does not exist" } 
    }
    query := val.query
    args := val.args

    rows, err := m.Conn.Query(query, args...)
    if err != nil {
        return result, err
    }
    defer rows.Close()

    for rows.Next() {
        types, tErr := rows.ColumnTypes()
        if tErr != nil {
            return result, tErr
        }

        values := make([]any, len(types))
        refs := make([]any, len(types))
        for i, t := range types {
            values[i] = reflect.New(t.ScanType())
            refs[i] = &values[i]
        }
        err = rows.Scan(refs...)
        if err != nil {
            return result, err
        }

        result = append(result, values)
    }

    return result, nil
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
    QueryMap map[string]MigratorQuery
    SortedQueryList []MigratorQuery 
}

type MigratorQuery struct {
    queryPath string
    number int
    query string
    args []any
}

// @NewMigrator 
// returns an instance of the Migrator struct with a provided connection reference.
func NewMigrator(conn *sql.DB) (Migrator, error) {
    m := Migrator{
        Conn: conn,
        QueryMap: make(map[string]MigratorQuery),
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
    if val, ok := m.QueryMap[queryName]; ok {
        val.args =args
        m.QueryMap[queryName] = val
        return nil
    }

    return &Err{ message: "queryName was not valid."} 
}

// TODO: Eventually this should be able to ingest a custom table via the QueryMap
func (m *Migrator) InitMigrationTable() (error) {
    query := m.QueryMap["util_queries/initial_schema"].query 
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
            splitName := strings.Split(cleanName[0], "_")
            num, err := strconv.Atoi(splitName[len(splitName)-1])
            if err != nil {
                return err
            }

            q := MigratorQuery{
                queryPath: (dir + "/" + cleanName[0]),
                number: num,
                query: string(data),
                args: make([]any, 0),
            }
            m.QueryMap[dir + "/" + cleanName[0]] = q
        }
    }
    return nil
}

// TODO: This could probably be done in a smarter way - brute force for now.
func (m *Migrator) MakeSortedQueryList() {
    res := []MigratorQuery{}
    for _, v := range m.QueryMap {
        res = append(res, v)
    }
    sort.Slice(res, func(i, j int) bool {
        return res[i].number < res[j].number
    })

    m.SortedQueryList = res
}
