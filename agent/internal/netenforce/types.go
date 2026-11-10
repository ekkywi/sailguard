package netenforce

type Spec struct {
	Mode        string
	Hash        string
	Version     int64
	SafetyAllow []Rule
	Rules       []Rule
}

type Rule struct {
	ID        string
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
	Source    string
}

