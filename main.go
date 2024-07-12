package migrator

import (
	"database/sql"
	"os"
	"strings"
    "strconv"
    "sort"
)

// UTIL FUNCTIONS
func pop(l *[]string) string {
    f := len(*l)
    rv := (*l)[f-1]
    *l = (*l)[:f-1]
    return rv
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

func NewMigrator(conn *sql.DB) (Migrator, error) {
    m := Migrator{
        Conn: conn,
        QueryMap: make(map[string]MigratorQuery),
    }

    // Attempt to create the base table if needed.
    err := m.InitMigrationTable()
    if err != nil {
        return Migrator{}, err 
    }

    return m, nil
}

func (m *Migrator) MigrateUp() (error) {
    tx, err := m.Conn.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var lastMigration int
    qErr := m.Conn.QueryRow("SELECT COALLESCE(0, migration_number) as num FROM migrations;").Scan(&lastMigration)
    if qErr != nil {
        return err
    }


    for i, mq := range m.SortedQueryList {
        if i+1 <= lastMigration {
            continue
        }
        tx.Exec(mq.query, mq.args...)
        tx.Exec("INSERT INTO migrations (migration_number) VALUES (?)", i+1)
    }

    if err = tx.Commit(); err != nil {
        return err
    }

    return nil
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
    query := "CREATE TABLE IF NOT EXISTS migrations (migration_number INTEGER PRIMARY KEY, created_at TIMESTAMP DEFAULT NOW());"
    _, err := m.Conn.Query(query)
    if err != nil {
        return err
    }
    return nil
}

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
    m.MakeSortedQueryList()
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
