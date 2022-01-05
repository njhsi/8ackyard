# 8ackyard
backup data smart and private

## usage
$ apt install exiftool # MUST
$ ./8ackyard index /mnt/media #only indexing
$ ./8ackyard index /mnt/media -b /mnt/backup #backup into meida type(audio, video, photo) and date

## notes
For this timebeing, this tool copied a lot from photoprism codes.
With some optimizations on:
- making exiftool stay open when indexing
- supporting audio
- guessing time from names
- copying file, and its stat/mtime..
- ...
