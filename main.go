package migrator
// TODO: MakeSortedQueryList should just be run before migrateup is called.

import (
	"database/sql"
	"os"
	"strings"
    "strconv"
    "sort"
    "fmt"
    "bufio"
)

// UTIL FUNCTIONS
func pop(l *[]string) string {
    f := len(*l)
    rv := (*l)[f-1]
    *l = (*l)[:f-1]
    return rv
}

func confirmUserAction(message string) bool {
    reader := bufio.NewReader(os.Stdin)
    fmt.Println(message)
    text, _ := reader.ReadString('\n')
    text = strings.TrimSuffix(text, "\n")
    if text == "y" {
        fmt.Println("Lets do it! :)")
        return true
    } 
    fmt.Println("You said no, good choice.")
    return false
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
    moveForward := confirmUserAction("Do you want to migrate up? [y/n] \n")
    if !moveForward {
        return nil
    }

    tx, err := m.Conn.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var lastMigration int
    rows, qErr := m.Conn.Query("SELECT COALESCE(max(migration_number), 0) as lastMigration FROM migrations;")
    if qErr != nil {
        return qErr
    }
    for rows.Next() {
        scanErr := rows.Scan(&lastMigration)
        if scanErr != nil {
            return scanErr
        }
    }

    for i, mq := range m.SortedQueryList {
        if i+1 <= lastMigration {
            continue
        }
        fmt.Println("Executing Migration#", i+1);
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
        fmt.Println("Migration Done - #", i+1)
    }

    if err = tx.Commit(); err != nil {
        return err
    }

    fmt.Println("All migrations done and commited")
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

// Removes all tables form a schema. Only used for quick testing/automation.
func (m *Migrator) MigrateDown(schemaName string) (error) {
    if schemaName == "" {
        schemaName = "public"
    }
    moveForward := confirmUserAction("Do you really want to migrate down? This will destroy all data and tables in your provided database connection. [y/n] \n")
    if !moveForward {
        return nil
    }
    areYouSure := confirmUserAction("You are really sure you want to migrate down? This will nuke your DB and is only for quickly tearing down a testing DB. [y/n] \n")
    if !areYouSure {
        return nil
    }
    migrateDownFunc := `
    CREATE OR REPLACE FUNCTION migrate_down(schema_name text)
    RETURNS VOID AS $$
    DECLARE r record;
    BEGIN
        FOR r IN SELECT * FROM pg_tables WHERE schemaname = schema_name LOOP
            EXECUTE 'DROP TABLE ' || r.tablename || ' CASCADE;';
        END LOOP;
   END; $$ language plpgsql;` 
   _, err := m.Conn.Exec(migrateDownFunc)
   if err != nil {
       return err
   }

   _, err = m.Conn.Exec("SELECT migrate_down($1)", schemaName)
   if err != nil {
       return err
   }

   _, err = m.Conn.Exec("DROP FUNCTION migrate_down;")
   if err != nil {
       return err
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
        return res[i].Number < res[j].Number
    })

    m.SortedQueryList = res
}
