# immich-timestamp-fix

After importing a bunch of photos into immich from google photos, I ended up with a whole lot of files with the wrong create date. over 6000 photos all sharing August 1 2023 as the date. Fixing this manually would have taken forever, but thankfully most of these photos had date and timestamps in the filenames.

So I opened up an AI assited editor and had it create this crude script to update the immich timestamp to what was in the filename for any files currently using August 1st 2023.

I imgaine this could work for other people in a similar situation.

