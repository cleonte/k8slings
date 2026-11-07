# config5: a Secret with plain text where base64 belongs

The values under `data` of a Secret have to be **base64**; the API server rejects anything else. That is an encoding, not encryption. If you would rather write the text itself, `stringData` takes plain text and the API server stores it under `data` for you.

The Secret must hold the username `app` and the password `s3cr3t pass!`. Make the manifest valid.
