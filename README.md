# Recursive Sum (recsum)

A little tool written to get more familiar with go channels.

Building `cmd/recsum.go` provides a binary that will generate hash checksum
output. It can be echo'd or written directly to a file.

The currently supported algorithms are `md5` `sha1` `sha256` and `sha512`

## Tool Usage

Setting `-w #` will change how many files are simultaneously read and sums'
calculated.

```
# ./recsum --help
Usage: ./recsum [OPTIONS] FILE...
version: v0.0.1

recsum is a tool for recursively generating hash sums

  -h string
    	Hash algorithm to use [md5, sha1, sha256, sha512] (default "sha256")
  -o string
    	Output file path (default "-")
  -v	Verbose
  -w int
    	Simultaneous workers (default 3)
```

## Package Usage

```go
outCh := make(chan *recsum.HashResult)

wg.Add(1)
go func() {
	defer wg.Done()
	for result := range outCh {
		if result.Error != nil {
			continue
		}
		fmt.Printf("%s  %s\n", result.Hash, result.Path)
	}
}()

recursor, _ := recsum.New("/etc", crypto.MD5, outCh, 3)
recursor.Walk()

close(outCh)
wg.Wait()
```

## Tool Example

```
# ls test/
test1.txt  test2.txt

# ./recsum -v -o test.sum test/
test/test1.txt completed in 476.98µs
test/test2.txt completed in 24.72636ms
Output written to 'test.sum'
Completed in 25.09019ms

# sha256sum -c test.sum 
test/test1.txt: OK
test/test2.txt: OK
```