# Frab 2 HackerTracker

This is a hacky application to take the JSON schedule and speakers from [frab](https://frab.github.io/frab/) and output JSON formatted for [HackerTracker](https://hackertracker.app/)

## Building

Run `make`

## Usage

Pass the base URL for the frab conference schedule with the `-frab` flag, and the required JSON files are created. You can use the `-save` flag to change the output directory, or the `-id` flag to offset the IDs from frab to HackerTracker.

Example:

```
./frab2ht -frab https://frab.toorcon.net/en/toorcon20
```
