# jsonparser - Fastest JSON parser for Go

## Benchmarks

There is 3 benchmark types, trying to simulate real-life usage for small, medium and large JSON payloads.
For each metric, the lower value is better. Values better then standard encoding/json marked as bold text.

Compared libraries:
* https://golang.org/pkg/encoding/json
* https://github.com/Jeffail/gabs
* https://github.com/bitly/go-simplejson
* https://github.com/antonholmquist/jason
* https://github.com/pquerna/ffjson
* https://github.com/buger/jsonparser

#### TLDR
If you want to skip next sections, winner is `jsonparser` (obviously benchmarks are biased :smirk:).
It is 3-9 times faster then standard `encoding/json` package (depending on payload size and usage), and almost infinitely (literally) better in memory consumption because it operate with data on byte level, and provide direct slice pointers.
Few allocations you see in benchmarks happen because type conversions.


#### Small payload

Each test should process 190 byte http log like json record.
It should read multiple fields.
https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_small_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 6173 | 880 | 18 |
| encoding/json interface{} | 7901 | 1521 | 38|
| Jeffail/gabs | 7836 | 1649 | 46 |
| bitly/go-simplejson | 8273 | 2241 | 36 |
| antonholmquist/jason | 20941 | 7237 | 101 |
| pquerna/ffjson | **3163** | **624** | **15** |
| buger/jsonparser | **714** | **4** | **2** |

Winners are ffjson and jsonparser, where jsonparser is 8.6x faster then encoding/json and 4.4x faster then ffjson. 
If you look at memory allocation, jsonparser have no rivals, as it makes no data copy and operate with raw []byte structures and pointers to it. 

#### Medium payload

Each test should process 2.4kb json record (based on Clearbit API).
It should read multiple nested fields and 1 array.

https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_medium_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 53251 | 1336 | 29 |
| encoding/json interface{} | 60781 | 10627 | 215 |
| Jeffail/gabs | 71547 | 11202 | 235 |
| bitly/go-simplejson | 67865 | 17187 | 220 |
| antonholmquist/jason | 70964 | 19013 | 247 |
| pquerna/ffjson | **19634** | **856** | **20** |
| buger/jsonparser | **11442** | **18** | **2** |

Pattern is clear, difference between ffjson and jsonparser in CPU is smaller, but memory consumption difference only grows.
gabs, go-simplejson and jason are based on encoding/json and map[string]interface{} and actually only helpers for unstructured JSON, their performance correlate with `encoding/json interface{}`, and they will skip next round.


#### Large payload

Each test should process 24kb json record (based on Discourse API)
It should read 2 arrays, and for each item in array get few fields.
Basically it means processing full JSON file.

https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_large_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 602245 | 8273 | 307 |
| encoding/json interface{} | 941123 | 215433 | 3395 |
| pquerna/ffjson | **287151** | **7792** | **298** |
| buger/jsonparser | **193601** | **120** | **32** |

Same patterns as at medium test. Both `ffjson` and `jsonparser` have own parsing code, and not depend on `encoding/json` or `interface{}`, thats one of the reasons why it so fast.

#### Conclusion

`jsonparser` is obvious winner, especially at memory allocation. 
`ffjson` goes next and looks really great, especially considering that it is almost drop-in replacement for `encoding/json`.
