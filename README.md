# go-pq-migrator
Migration manager for postgres in Golang. 

`go get github.com/skye-lopez/go-pq-migrator`

Feel free to make any feature requests, or submit a PR.

(does not support the raw database/sql but would be an easy fork)

## Current State
v1.0.0 : This is a very rough beta: USE WITH CAUTION

The roadmap still has some critical things to push.

### Roadmap
- Better tests
- Better errors 
- Optional CLI confirmation

# Usage

Before we get started some context on usage

API can be found at the end

### File format
To get proper use of this package your queries should live in a folder (can be nested) with your migration files always following an ordered state. If a file does not fit the format it will be skipped.

The official format is: `{anything}_{num}.sql`

```
|-- /migrations (can be any folder name)
   |-- initial_schema_001.sql
   |-- awesome_name_002.sql
   |-- /nested_migrations
      |-- somename_3.sql
```

This would produce the following order of operations:
```
initial_schema_001.sql->awesome_name_002.sql->somename_3.sql
```

### This creates a new table in your db. 

This script will by default create the following table:
```sql
CREATE TABLE migrations (migration_number INT PRIMARY KEY, created_at TIMESTAMP DEFAULT NOW());
```

Whenever a migration file is executed it will be added to the history log of this table. This is used as a reference as to what migration number should be executed next.

### Code example
```golang
package main

import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/skye-lopez/go-pq-migrator"
)

func main() {
    db := yourDbConnCode() // Connect to sql however you want

    m, err := migrator.NewMigrator(db)
    if err != nil {
        // handle your error, this likely means there is an issue with db *sql.DB provided.
    }

    err = m.AddQueriesToMap("migrations") // this can be any folder name with your queries

    // If your migration requires arguments to be executed you can pass them this way
    // See API for more info on string formatting
    var args []any
    err = m.AddArgsToQuery("migrations/initial_schema_001", args)

    err = m.MigrateUp() // migrates up, will have a CLI confirmation.

    err = m.MigrateDown("schemaName") // also supports destroying all tables in a schema.
}
```

## API
### @Migrator.NewMigrator
Constructor that returns a Migrator class. It requires a valid sql.DB and returns a new Migrator struct.
```golang
// NewMigrator(conn *sql.DB) => (Migrator, error)

m, err := NewMigrator(conn)
```

### @Migrator.AddQueriesToMap
This function takes in a `dirPath` to the folder containing your migration queries. It can be nested. Each individual file must follow the correct format - `{anything}_{number}.sql ex; name_001.sql` or it will be skipped.
```
|-- /migrations
   |-- initial_schema_001.sql
   |-- awesome_name_002.sql
   |-- /nested_migrations
      |-- somename_3.sql
```
```
m.AddQueriesToMap("migrations")

Migrator.QueryMap will now have the following KV pairs
migrations/initial_schema_001=>MigratorQuery{...}
migrations/awesome_name_002=>MigratorQuery{...}
migrations/somename_3=>MigratorQuery{...}
```

### @Migrator.MigrateUp
Runs migrations in order from last performed migration to newest found via `AddQueriesToMap`. This required user input to continue via the cli.

Note: There will be an option to remove the cli confirmation soon.
```golang
m.MigrateUp()
```

### @Migrator.MigrateDown
Destroys all tables found in a provided schema. This has 2 levels of user input confirmation via the cli.
```golang
m.MigrateDown("schemaName")
```
