package migrator

import (
	"database/sql"
	"os"
	"strings"
    "strconv"
    "sort"
    "fmt"
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
    QueryPath string
    Number int
    Query string
    Args []any
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

// NOTE: This doesn't capture any kind of rowResults from the query in question; maybe thats wanted at some point and would be a quick refactor.
func (m *Migrator) MigrateUp() (error) {
    tx, err := m.Conn.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var lastMigration int
    qErr := m.Conn.QueryRow("SELECT COALLESCE(0, migration_number) as num FROM migrations;").Scan(&lastMigration)
    // TODO: Remove this or turn on optional logging (and add more logging)
    fmt.Println("Last Migration:", lastMigration)
    if qErr != nil {
        return err
    }

    for i, mq := range m.SortedQueryList {
        if i+1 <= lastMigration {
            continue
        }
        if len(mq.Args) >= 1 {
            _, err := tx.Exec(mq.Query, mq.Args...)
            if err != nil {
                return err
            }
        } else {
            _, err := tx.Exec(mq.Query)
            if err != nil {
                return err
            }
        }
        _, err := tx.Exec("INSERT INTO migrations (migration_number) VALUES ($1)", i+1)
        if err != nil {
            return err
        }
    }

    if err = tx.Commit(); err != nil {
        return err
    }

    return nil
}

// TODO: Ideally this would also be able to take a migration number as 
// if the pathName is highly nested this could become cumbersome...
// TODO: Also post that decision we need to make sure migrator.QueryMap and migrator.SortedQueryList are in sync.
func (m *Migrator) AddArgsToQuery(queryName string, args []any) (error) {
    if val, ok := m.QueryMap[queryName]; ok {
        val.Args = args
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

// TODO: Validate filenames, current accepted format is: {anything}_{MigrationNumber}.sql ; example: initial_schema_001.sql 
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
                QueryPath: (dir + "/" + cleanName[0]),
                Number: num,
                Query: string(data),
                Args: make([]any, 0),
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
        return res[i].Number < res[j].Number
    })

    m.SortedQueryList = res
}
