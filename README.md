# storage-validator
Generate and validate directory checksum against all nested files

### Execution flags

|Flag|Env. variable|Default value|Description|
|:----|:----|:---|:---|
|d|D||Working directory path|
|m|M||Operation mode (init/validate/reset)|

### Usage

Init directory:  
`go run cmd/storage-validator.go -d /path/to/dir -m init`

Validate initialized directory:  
`go run cmd/storage-validator.go -d /path/to/dir -m validate`

Reset storage validator data (doesn't affect original directory contents):  
`go run cmd/storage-validator.go -d /path/to/dir -m reset`
