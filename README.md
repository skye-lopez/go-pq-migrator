# go-pq-migrator
Migration manager for postgres in Golang

# Roadmap/Status
Documentation and full build coming in the next few days.

What it is supposed to accomplish:

given a folder(can be nested) of (name formatted)migration .sql files, run them in proper order while keeping context of a prior migration.

Some tools to help with the implementation of the former

And if you want to MigrateDown() [DELETE YOUR DB] it will do that too. This can be usefull for testing.
