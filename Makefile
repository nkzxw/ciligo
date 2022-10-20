.PHONY: build pack test lint clean

build:
	scripts/build.sh

start:
	scripts/start.sh

stop:
	scripts/stop.sh

pack:
	scripts/build.sh pack

test:
	scripts/test.sh

lint:
	scripts/lint.sh

clean:
	scripts/clean.sh
