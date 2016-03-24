# Alternative JSON parser for Go

It does not require you to know the structure of the payload (eg. create structs), and allows accessing fields by providing the path to them. It is up to **7 times faster** then standard `encoding/json` package (depending on payload size and usage), **allocates almost no memory**. See benchmarks below.

## Rationale
Originally I made this for a project that relies on a lot of 3rd party APIs that can be unpredictable and complex.
I love simplicity and prefer to avoid external dependecies. `encoding/json` requires you to know exactly your data structures, or if you prefer to use `map[string]interface{}` instead, it will be very slow and hard to manage.
I investigated what's on the market and found that most libraries are just wrappers around `encoding/json`, there is few options with own parsers (`ffjson`, `easyjson`), but they still requires you to create data structures.
Let's be honest, JSON is not the hardest format to parse, so i wrote one that focuses on simplicity and performance and leverage Go datastructures. 

## Example
For the given JSON our goal is to extract the user's full name, number of github followers and avatar.

```go
import "github.com/buger/jsonparser"

...

data := []byte(`{
  "person": {
    "name": {
      "first": "Leonid",
      "last": "Bugaev",
      "fullName": "Leonid Bugaev"
    },
    "github": {
      "handle": "buger",
      "followers": 109
    },
    "avatars": [
      { "url": "https://avatars1.githubusercontent.com/u/14009?v=3&s=460", "type": "thumbnail" }
    ]
  },
  "company": {
    "name": "Acme"
  }
}`)

// You can specify key path by providing arguments to Get function
jsonparser.Get(data, "person", "name", "fullName")

// There is `GetNumber` and `GetBoolean` helpers if you exactly know key data type
jsonparser.GetNumber(data, "person", "github", "followers")

// When you try to get object, it will return you []byte slice pointer to data containing it
// In `company` it will be `{"name": "Acme"}`
jsonparser.Get(data, "company")

// If the key doesn't exist it will throw an error
var size float64
if value, _, err := jsonparser.GetNumber(data, "company", "size"); err != nil {
  size = value
}

// You can use `ArrayEach` helper to iterate items
jsonparser.ArrayEach(data, func(value []byte, dataType int, offset int, err error) (err error) {
	fmt.Println(jsonparser.Get(value, "url"))
}, "person", "gravatar", "avatars")
```

## Reference

Library API is really simple. You just need the `Get` method to perform any operation. The rest is just helpers around it.

You also can view API at [godoc.org](https://godoc.org/github.com/buger/jsonparser)


### **`Get`**
```
func Get(data []byte, keys ...string) (value []byte, dataType int, offset int, err error)
```
Receives data structure, and key path to extract value from.

Returns:
* `value` - Pointer to original data structure containing key value, or just empty slice if nothing found or error
* `dataType` - 	Can be: `NotExist`, `String`, `Number`, `Object`, `Array`, `Boolean` or `Null`
* `offset` - Offset from provided data structure where key value ends. Used mostly internally, for example for `ArrayEach` helper.
* `err` - If the key is not found or any other parsing issue, it should return error. If key not found it also sets `dataType` to `NotExist`

Accepts multiple keys to specify path to JSON value (in case of quering nested structures).
If no keys are provided it will try to extract the closest JSON value (simple ones or object/array), useful for reading streams or arrays, see `ArrayEach` implementation.

### **`GetBoolean`** and **`GetNumber`**
```
func GetBoolean(data []byte, keys ...string) (val bool, offset int, err error)

func GetNumber(data []byte, keys ...string) (val float64, offset int, err error)
```
If you know the key type, you can use the helpers above. Returns same arguments as `Get` except `dataType`.
If key data type do not match, it will return error.

### **`ArrayEach`**
```
func ArrayEach(data []byte, cb func(value []byte, dataType int, offset int, err error), keys ...string)
```
Needed for iterating arrays, accepts a callback function with the same return arguments as `Get`.


## What makes it so fast?
* It does not rely on `encoding/json`, `reflection` or `interface{}`, the only real package dependency is `bytes`.
* Operates with JSON payload on byte level, providing you pointers to the original data structure: no memory allocation.
* No automatic type conversions, by default everything is a []byte, but it provides you value type, so you can convert by yourself (there is few helpers included).
* Does not parse full record, only keys you specified


## Benchmarks

There are 3 benchmark types, trying to simulate real-life usage for small, medium and large JSON payloads.
For each metric, the lower value is better. Time/op is in nanoseconds. Values better than standard encoding/json marked as bold text.
Benchmarks run on standard Linode 1024 box.

Compared libraries:
* https://golang.org/pkg/encoding/json
* https://github.com/Jeffail/gabs
* https://github.com/bitly/go-simplejson
* https://github.com/antonholmquist/jason
* https://github.com/mreiferson/go-ujson
* https://github.com/ugorji/go/codec
* https://github.com/pquerna/ffjson
* https://github.com/mailru/easyjson
* https://github.com/buger/jsonparser

#### TLDR
If you want to skip next sections we have 2 winner: `jsonparser` and `easyjson`.
`jsonparser` is up to 7 times faster then standard `encoding/json` package (depending on payload size and usage), and almost infinitely (literally) better in memory consumption because it operates with data on byte level, and provide direct slice pointers. The few allocations you see in benchmarks happen because of type conversions.
`easyjson` wins in CPU in small and medium tests, and loose in large. Frankly i'm impressed with this package, and it is remarkable results considering that it is almost drop-in replacement for `encoding/json` (require some code generation).

It's hard to fully compare `jsonparser` and `easyjson` (or `ffson`), they a true parsers and fully process record, unlike `jsonparser` which parse only keys you specified.

If you searching for replacement of `encoding/json` while keeping structs, `easyjson` is an amazing choise. If you want to process dynamic JSON, have memory constrains, or more control over your data you should try `jsonparser`.

`jsonparser` performance heavily depends on usage, and it works best when you do not need to process full record, only some keys. The more calls you need to make, the slower it will be, in contrast `easyjson` (or `ffjson`, `encoding/json`) parser record only 1 time, and then you can make as many calls as you want.

With great power comes great responsibility! :)


#### Small payload

Each test processes 190 bytes of http log as a JSON record.
It should read multiple fields.
https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_small_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 7577 | 880 | 18 |
| encoding/json interface{} | 9030 | 1521 | 38|
| Jeffail/gabs | 10153 | 1649 | 46 |
| bitly/go-simplejson | 9993 | 2241 | 36 |
| antonholmquist/jason | 27745 | 7237 | 101 |
| github.com/ugorji/go/codec | 8604 | 2176 | 31 |
| mreiferson/go-ujson | **7185** | **1409** | 37 |
| pquerna/ffjson | **3816** | **624** | **15** |
| mailru/easyjson | **1965** | **192** | **9** |
| buger/jsonparser | **2131** | **4** | **2** |

Winners are ffjson, easyjson and jsonparser, where jsonparser is 3.5x faster then encoding/json and 1.8x faster then ffjson, but slightly slower then easyjson.
If you look at memory allocation, jsonparser has no rivals, as it makes no data copy and operates with raw []byte structures and pointers to it.

#### Medium payload

Each test processes a 2.4kb JSON record (based on Clearbit API).
It should read multiple nested fields and 1 array.

https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_medium_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 59019 | 1336 | 29 |
| encoding/json interface{} | 79315 | 10627 | 215 |
| Jeffail/gabs | 82896 | 11202 | 235 |
| bitly/go-simplejson | 90586 | 17187 | 220 |
| antonholmquist/jason | 93233 | 19013 | 247 |
| github.com/ugorji/go/codec | 115904 | 6712 | 152 |
| mreiferson/go-ujson | **57038** | 11547 | 270 |
| pquerna/ffjson | **20433** | **856** | **20** |
| mailru/easyjson | **10642** | **336** | **12** |
| buger/jsonparser | **17158** | **18** | **2** |

The difference between ffjson and jsonparser in CPU usage is smaller, while the memory consumption difference is growing. On the other hand `easyjson` shows remarkable performance for medium payload.

`gabs`, `go-simplejson` and `jason` are based on encoding/json and map[string]interface{} and actually only helpers for unstructured JSON, their performance correlate with `encoding/json interface{}`, and they will skip next round.
`go-ujson` while have its own parser, shows same performance as `encoding/json`, also skips next round. Same situation with `ugorji/go/codec`, but it showed unexpectedly bad performance for complex payloads.


#### Large payload

Each test processes a 24kb JSON record (based on Discourse API)
It should read 2 arrays, and for each item in array get a few fields.
Basically it means processing a full JSON file.

https://github.com/buger/jsonparser/blob/master/benchmark/benchmark_large_payload_test.go

| Library | time/op | bytes/op | allocs/op |
| --- | --- | --- | --- | --- |
| encoding/json struct | 720045 | 8272 | 307 |
| encoding/json interface{} | 1228126 | 215425 | 3395 |
| pquerna/ffjson | **315777** | **7792** | **298** |
| mailru/easyjson | **161000** | **6992** | **288** |
| buger/jsonparser | **94660** | **88** | **30** |

`jsonparser` now is a winner, but do not forget that it is way more lighweight parser then `ffson` or `easyjson`, and they have to parser all the data, while `jsonparser` parse only what you need. All `ffjson`, `easysjon` and `jsonparser` have their own parsing code, and does not depend on `encoding/json` or `interface{}`, thats one of the reasons why they are so fast. `easyjson` also use a bit of `unsafe` package to reduce memory consuption (in theory it can lead to some unexpected GC issue, but i did not tested enough)

## Questions and support

All bug-reports and suggestions should go though Github Issues.
If you have some private questions you can send them directly to me: leonsbox@gmail.com

## Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Added some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create new Pull Request

## Development

All my development happens using Docker, and repo include some Make tasks to simplify development.

* `make build` - builds docker image, usually can be called only once
* `make test` - run tests
* `make fmt` - run go fmt
* `make bench` - run benchmarks (if you need to run only single benchmark modify `BENCHMARK` variable in make file)
* `make profile` - runs benchmark and generate 3 files-  `cpu.out`, `mem.mprof` and `benchmark.test` binary, which can be used for `go tool pprof`
* `make bash` - enter container (i use it for running `go tool pprof` above)
