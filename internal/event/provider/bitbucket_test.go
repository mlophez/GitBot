package provider

import (
	"os"
	"testing"
)

func TestGetAuthor(t *testing.T) {
	bitbucket := NewBitbucketProvider(os.Getenv("BITBUCKET_BEARER_TOKEN"))

	bitbucket.getAuthor("https://api.bitbucket.org/2.0/users/%7B981c5124-f0f0-4b95-b543-82a33d53aabc%7D")

}
