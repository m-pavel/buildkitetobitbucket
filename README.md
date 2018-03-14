# buildkitetobitbucket
Buildkite trigger for bitbucket.org. Usefull when you need to trigger pipeline by changes in the different bitbucket repository that configured in buildkite.

# Setup
```
go get -u m-pavel/buildkitetobitbucket
```
Configure webhook on bitbucket by adding new one.
```
http://yourserver:8080/v1/start/<org>/<pipeline>/<pipeline branch>
```

