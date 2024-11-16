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

create_test_dir:
    @ rm -rf var/storeme
    @ mkdir var/storeme
    @ "hello" | save var/storeme/test.txt
