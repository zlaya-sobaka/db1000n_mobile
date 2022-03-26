# Death by 1000 needles

See [Docs](https://arriven.github.io/db1000n)

This is a simple distributed load generation client written in go.
It is able to fetch simple json config from a local or remote location.
The config describes which load generation jobs should be launched in parallel.
There are other existing tools doing the same kind of job.
I do not intend to copy or replace them but rather provide a simple open source alternative so that users have more options.
Feel free to use it in your load tests (wink-wink).

The software is provided as is under no guarantee.
I will update both the repo and this doc as I go during following days (date of writing this is 26th of February 2022, third day of Russian invasion into Ukraine).

[Gitlab mirror](https://gitlab.com/db1000n/db1000n.git)

go get -u github.com/fyne-io/fyne-cross

go get fyne.io/fyne/v2/cmd/fyne
go install fyne.io/fyne/v2/cmd/fyne

fyne release -os android -appID zlaya.sobaka.db1000n_mobile -appVersion 1.0 -appBuild 1 -keyStore ./my-release-key.keystore -icon ./Icon.png
