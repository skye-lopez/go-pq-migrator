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

type Migrator struct {
    Conn *sql.DB
    QueryMap map[string]string
}

// @NewMigrator 
// returns an instance of the Migrator struct with a provided connection reference.
func NewMigrator(conn *sql.DB) Migrator {
    return Migrator{
        Conn: conn,
        QueryMap: make(map[string]string),
    }
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
[queryOne]=>queryOneContents, [nested/queryTwo]=>queryTwoContents, [nested/nestedAgain/queryThree]=>queryThreeContents
*/

// TODO: We should skip files that are not .sql
// TODO: for some reason it seems buggy if we pass "./q" instead of "q" as the root for example... need to test more cases
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
            m.QueryMap[dir + "/" + cleanName[0]] = string(data)
        }
    }
    return nil
}
