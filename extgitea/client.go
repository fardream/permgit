package extgitea

import "code.gitea.io/sdk/gitea"

func version() string {
	return gitea.Version()
}
