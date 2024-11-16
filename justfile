set shell := ["nu", "-c"]

run *a: compile
    @ ./bin/recycle {{a}}

compile:
    @ rm -rf bin
    @ mkdir bin
    @ go build -o bin/recycle

install: compile
    @ mkdir ~/.recycle
    @ mv bin/recycle ~/.recycle/recycle
