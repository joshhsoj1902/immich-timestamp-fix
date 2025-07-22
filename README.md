# immich-timestamp-fix

After importing a bunch of photos into immich from google photos, I ended up with a whole lot of files with the wrong create date. over 6000 photos all sharing August 1 2023 as the date. Fixing this manually would have taken forever, but thankfully most of these photos had date and timestamps in the filenames.

So I opened up an AI assited editor and had it create this crude script to update the immich timestamp to what was in the filename for any files currently using August 1st 2023.

I imgaine this could work for other people in a similar situation.

If you do want to use this, I recomend commenting out the update call to ensure it'll do what you want.

Set the day you need fixed here: https://github.com/joshhsoj1902/immich-timestamp-fix/blob/main/main.go#L32

Add regex matching here: https://github.com/joshhsoj1902/immich-timestamp-fix/blob/main/main.go#L139

Run it like this:

```
IMMICH_API_KEY=<API KEY> IMMICH_API_URL=http://immich.example.ca/api go run main.go
```
