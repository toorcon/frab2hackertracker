SOURCES := main.go hackertracker-structs.go frab-structs.go

.PHONY: all fmt clean install

all: frab2ht

frab2ht: $(SOURCES)
	go build -o $@ $(SOURCES)

fmt:
	gofmt -s -w -l .

install: $(SOURCES)
	go install

clean:
	rm -r frab2ht