**Description**: What this PR does

**Benchmark before change**:

**Benchmark after change**:


For running benchmarks use:
```
go test -test.benchmem -bench JsonParser ./benchmark/ -benchtime 5s -v
# OR
make bench (runs inside docker)
```