Varasto has APIs so external systems can interact with the data in Varasto. There are two
distinct API families:

1. Traditional, REST-based API (with a write-side twist, which will be explained)
2. Realtime push-based data API

!!! tip 
	You can even mix the two: use the push-based API to receive triggers, but still use the
	traditional API to do further processing.


Traditional API
---------------

### Architecture

Our traditional API is broken into two parts:

| Purpose          | Name & docs |
|------------------|-------------|
| For reading data | [REST API](../../stoserver/stoservertypes/rest_endpoints.md) |
| For writing data | [Commands](../../stoserver/stoservertypes/commands.md)       |

This separation that we're doing is known as [CQRS](https://martinfowler.com/bliki/CQRS.html) -
extremely short summary:

> different model to update information than the model you use to read information


### Example code

With our API reference you can use other languages as well, but we have first-class
language support for Go.

This code sample renames a directory and demoes both read and write endpoints:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func renameDir(ctx context.Context, dirId string, newDirName string) error {
	// read Varasto client's configuration file so we know address of Varasto server
	// and the auth token
	conf, err := stoclient.ReadConfig()
	if err != nil {
		return err
	}

	// use the REST API to get directory's details. also includes additional detail like
	// directory's parents etc. for demo's sake (we don't actually need this to rename the directory)
	dirDetails := stoservertypes.DirectoryOutput{}
	if _, err := ezhttp.Get(
		ctx,
		conf.UrlBuilder().GetDirectory(dirId),
		ezhttp.RespondsJson(&dirDetails, false),
		ezhttp.Client(conf.HttpClient()),
	); err != nil {
		return err
	}

	fmt.Printf(
		"directory old name = %s, type = %s\n",
		dirDetails.Directory.Name,
		dirDetails.Directory.Type)

	// rename the directory
	if err := conf.CommandClient().Exec(ctx, &stoservertypes.DirectoryRename{
		Id:   dirId,
		Name: newDirName,
	}); err != nil {
		return err
	}

	return nil
}

// use example:
//   $ ./renamedir <dirId> <newName>
func main() {
	if err := renameDir(context.TODO(), os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

```


Realtime push-based data API
----------------------------

!!! info
	This is for paid users only. It enables external systems to receive data changes in realtime.

Changes in Varasto are published as events that subscribers can monitor in realtime. The
event stream has exactly-once delivery semantics so you can trust to receive all the events
even if your subscriber crashes/goes offline for amounts of time.

This is the same fabric that our high-availability multi-master clustering is built on.

Contact us for more info.
