# A port of Microsoft Php Tolerant Parser to golang
This is a port of the [tolerant-php-parser](https://github.com/Microsoft/tolerant-php-parser) (mstpp) from Microsoft implemented in [golang](http://golang.org/). You can learn in mstpp github page about the design of this parser.

### Why?
This is my first project in Go and was an excuse to learn the language. So it is not by any means idiomatic/good go code yet. 

### What is the current status?
- Lexer: 93 tests pass / 4 fail
- Parser: 702 tests pass
 

### Trying gphp

This project was tested using Go 1.9.

Install gphp in your `$GOPATH`. Usually this means:
```bash
# clone the project
$ git clone --recursive git@github.com:emilioastarita/gphp.git $GOPATH/src/github.com/emilioastarita/gphp/
# install deps
$ go get -u -v github.com/yudai/gojsondiff/formatter
```

Lex and parse with cmd utility gphp
```bash
cd $GOPATH/src/github.com/emilioastarita/gphp/debug

# tokenize a php file with gphp
go run gphp.go scan some-file.php

# print ast 
go run gphp.go parse some-file.php

# Compare output with mstpp ast/tokens
go run gphp.go compare parse some-file.php
# or 
go run gphp.go compare scan some-file.php
```

Also is useful to pass a directory instead of a file and gphp will recurively will parse, scan or compare all php files inside it.  


`mstpp` is used as source of truth so there is also a `debug.php` that uses `mstpp` mainly for test cases generation. 

```bash
# generating all .tree cases with mstpp
php debug.php gencase-parser ../parser/cases/
# generating all .tokens cases with mstpp
php debug.php gencase-tokens ../parser/cases/
```


### Important missing parts:
- Double quote/backtick/heredoc strings are implemented but could work a little hacky in some cases. mstpp is using internal php functions `token_get_all_nl` for tokenization so is obviusly more robust. Anyway most tests are passing. If you find some weird case please provide a minimal test case.
- Diagnosis tools
- More idiomatic go
