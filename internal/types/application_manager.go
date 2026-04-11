package types

type AppManager interface {
	List() ([]Application, error)
	Lock(app Application, targetBranch string, prID int) error
	Unlock(app Application) error
}
