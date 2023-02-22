# [api.sonurai.com](https://api.sonurai.com)

## Running locally
To run the API locally, you will need to have `gcloud` must be installed and configured.

Once you do, run `gcloud auth application-default login` to set up the credentials.

Next, set the `PROJECT_ID` and `PORT` environment variables.

Finally, start the server with `go run ./cmd/app/`.

## Image updater
The image updater function retrieves wallpapers from the Bing Wallpapers API for different locales.

To be added to the database (Firestore),
there must be a 1920x1200 image available and not already exist in the database.

Next, the images are annotated using the Google Vision API.
This allows wallpapers to be searched by these labels in the future and provide better SEO.

For non-English locales, the descriptions are translated to English using the Google Translate API.
If a previously stored wallpaper appears again but with a new English description,
the description is replaced but the original date is kept.
The function also splits the description to retrieve the title and copyright information.
