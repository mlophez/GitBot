package application

import (
	"fmt"
	"testing"

	AppArgoCDAdapter "github.com/MLR96/argocd-bot/internal/application/adapters/argocd"
	AppService "github.com/MLR96/argocd-bot/internal/application/service"
)

//func TestFindAppsFilesChanged(t *testing.T) {
//	//manager := ApplicationManager{}
//	//repoUrl := "https://bitbucket.org/firmapro/platform-poc.git"
//	//filesChanged := []string{"/app/overlays/dev/hola.cfg"}
//
//	adapter, err := application.NewApplicationArgoCDAdapter()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	apps, err := adapter.Find()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	for _, app := range apps {
//		fmt.Printf("application:\t %s, %t\n", app.Name, app.Locked)
//	}
//}

func TestLock(t *testing.T) {
	repoUrl := "https://bitbucket.org/firmapro/platform-poc.git"
	filesChanged := []string{"/app/overlays/dev/hola.cfg"}

	adapter := AppArgoCDAdapter.New()
	svc := AppService.New(adapter)

	apps, err := svc.FindAppsForEvent(repoUrl, filesChanged)
	if err != nil {
		t.Fatal(err)
	}

	for _, app := range apps {
		fmt.Printf("application:\t %s, %t\n", app.Name, app.Lock.Locked)
		//newApp, err := svc.LockApp(app, "test")
		newApp, err := svc.UnlockApp(app)
		//adapter.Clean(app)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("application:\t %s\n", newApp.Branch)
	}

	//if len(apps) == 0 {
	//	return
	//}

	//app := apps[0]

	//newApp, err := manager.unlock(app)

	//fmt.Printf("application:\t %s\n", newApp.Spec.Source.TargetRevision)
}
