# api.sonurai.com

## Running locally
`gcloud` must be installed and configured.

Then run `gcloud auth application-default login` to set up the credentials.

Set the `PROJECT_ID` and `PORT` environment variables.

Finally, run `go run ./cmd/app/` to start the server.

## image updater

This function requests wallpapers for different countries using the Bing Wallpapers API.

Wallpapers get added if both the 1920x1080 and the 1920x1200 images exist and are not yet in the database (Firestore).

For non-English countries, the descriptions are translated into English using the Google Translate API.

Finally, the images are annotated using the Google Vision API. This allows wallpapers to be searched by these labels in
the future and provide better SEO.

If previously stored, translated English wallpapers appear again but in English, then the descriptions are replaced,
but the original date is kept.

There is also splitting of the description to get the title and copyright.

