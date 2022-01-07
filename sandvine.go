package sandvine

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ApogeeNetworking/gonetssh"
	"github.com/ApogeeNetworking/gonetssh/universal"
)

type SRPCmds struct {
	SvAdmDbStatus      string
	SvAdmDbTransStatus string
	SvCLI              string
	DbStatus           string
	JbossServRe        string
	JbossServWrapperRe string
	PgsqlServRe        string
	PgSvStat           string
	PgSvSub            string
	PgSvStatShort      string
	PgSvSubShort       string
	StartDb            string
	StopDb             string
	StartAppServ       string
	StopAppServ        string
	SwitchUser         string
	SwitchUserPg       string
	CntrlD             string
	PgErr              string
	Quit               string
}

var (
	SrpCmds  SRPCmds
	contains = strings.Contains
	replace  = strings.ReplaceAll
	split    = strings.Split
)

func init() {
	SrpCmds = SRPCmds{
		SvAdmDbStatus:      "show service database status",
		SvAdmDbTransStatus: "show service database transaction-id status",
		SvCLI:              "svcli",
		DbStatus:           "show system services",
		JbossServRe:        `JBoss\sApplication\sServer\s+(.*)`,
		JbossServWrapperRe: `JBoss\sApplication\sServer\sWrapper\s+(.*)`,
		PgsqlServRe:        `PostgreSQL\s+(.*)`,
		PgSvStat:           "postgres --single -D /usr/local/pgsql/data sv_stat",
		PgSvSub:            "postgres --single -D /usr/local/pgsql/data sv_sub",
		PgSvStatShort:      "psql sv_stat svadmin",
		PgSvSubShort:       "psql sv_sub svadmin",
		StartDb:            "start service database",
		StopDb:             "stop service database",
		StartAppServ:       "start service application-server",
		StopAppServ:        "stop service application-server",
		SwitchUser:         "sudo su -",
		SwitchUserPg:       "su - pgsql",
		CntrlD:             "\x04",
		PgErr:              "\"postmaster.pid\" already exists",
		Quit:               "\x5C\x71",
	}
}

type Service struct {
	Client universal.Device
}

// NewService ...
func NewService(host, user, pass string) *Service {
	c, _ := gonetssh.NewDevice(
		host, user, pass, "", gonetssh.DType.SVSRP3000,
	)
	return &Service{Client: c}
}

// SrpDbStatus ...
type SrpDbStatus struct {
	CmdResults []string `json:"cmdResults"`
	OneHndrdMn int64    `json:"oneHndrdMn"`
	FiveHndrMn int64    `json:"fiveHndrdMn"`
	OneBn      int64    `json:"oneBn"`
	OneFiveBn  int64    `json:"oneFiveBn"`
	TwoBn      int64    `json:"twoBn"`
	DbError    bool     `json:"dbError"`
}

func (s *Service) GetSvAdmDbStatus() (SrpDbStatus, error) {
	var dbStatus SrpDbStatus
	out1, _ := s.Client.SendCmd(SrpCmds.SvAdmDbStatus)
	dbStatus.CmdResults = append(dbStatus.CmdResults, out1)
	if contains(out1, "Unable to connect to database") {
		dbStatus.DbError = true
		return dbStatus, errors.New("err: unable to connect to database")
	}
	out2, _ := s.Client.SendCmd(SrpCmds.SvAdmDbTransStatus)
	dbStatus.CmdResults = append(dbStatus.CmdResults, out2)
	hndrdMnRe := regexp.MustCompile(`GreaterThan100Million:\s(\S+)`)
	fiveHndrdMnRe := regexp.MustCompile(`GreaterThan500Million:\s(\S+)`)
	oneBnRe := regexp.MustCompile(`GreaterThan1Billion\s+:\s(\S+)`)
	one5BnRe := regexp.MustCompile(`GreaterThan1.5Billion:\s(\S+)`)
	twoBnRe := regexp.MustCompile(`GreaterThan2Billion\s+:\s(\S+)`)
	hndrdMnMatches := hndrdMnRe.FindStringSubmatch(out2)
	fiveHndrdMnMatches := fiveHndrdMnRe.FindStringSubmatch(out2)
	oneBnMatches := oneBnRe.FindStringSubmatch(out2)
	one5BnMatches := one5BnRe.FindStringSubmatch(out2)
	twoBnMatches := twoBnRe.FindStringSubmatch(out2)

	hndrdMn, _ := strconv.ParseInt(replace(hndrdMnMatches[1], ",", ""), 10, 64)
	fiveHndrdMn, _ := strconv.ParseInt(replace(fiveHndrdMnMatches[1], ",", ""), 10, 64)
	oneBn, _ := strconv.ParseInt(replace(oneBnMatches[1], ",", ""), 10, 64)
	onefiveBn, _ := strconv.ParseInt(replace(one5BnMatches[1], ",", ""), 10, 64)
	twoBn, _ := strconv.ParseInt(replace(twoBnMatches[1], ",", ""), 10, 64)
	dbStatus.OneHndrdMn = hndrdMn
	dbStatus.FiveHndrMn = fiveHndrdMn
	dbStatus.OneBn = oneBn
	dbStatus.OneFiveBn = onefiveBn
	dbStatus.TwoBn = twoBn
	return dbStatus, nil
}

type DbServiceStatus struct {
	JBossAppServer        string
	JBossAppServerWrapper string
	PgSQL                 string
}

// Trim Line for All MultiSpaced White Space for easier parsing
func (s *Service) trimWS(text string) string {
	t := replace(text, "[", "")
	t = replace(t, "]", "")
	tsRe := regexp.MustCompile(`\s+`)
	return tsRe.ReplaceAllString(t, " ")
}

func (s *Service) ShowDbServStatus() (DbServiceStatus, error) {
	out, err := s.Client.SendCmd(SrpCmds.DbStatus)
	jbossRe := regexp.MustCompile(SrpCmds.JbossServRe)
	jbossWrapperRe := regexp.MustCompile(SrpCmds.JbossServWrapperRe)
	pgsqlRe := regexp.MustCompile(SrpCmds.PgsqlServRe)
	jbossMatches := jbossRe.FindStringSubmatch(out)
	jbossWrapperMatches := jbossWrapperRe.FindStringSubmatch(out)
	pgsqlMatches := pgsqlRe.FindStringSubmatch(out)
	var dbStatus = DbServiceStatus{
		JBossAppServer:        split(s.trimWS(jbossMatches[1]), " ")[1],
		JBossAppServerWrapper: split(s.trimWS(jbossWrapperMatches[1]), " ")[1],
		PgSQL:                 split(s.trimWS(pgsqlMatches[1]), " ")[1],
	}
	return dbStatus, err
}

func (s *Service) TriageSrp() []string {
	dbStatus, _ := s.GetSvAdmDbStatus()
	// Check for Initial DB Errors
	// This will Dictate which Methods used for Vacuum of DB
	switch {
	case dbStatus.DbError:
		return s.LongVacuum()
	case dbStatus.OneBn > 0 || dbStatus.OneFiveBn > 0 || dbStatus.TwoBn > 0:
		return s.ShortVacuum()
	}
	return []string{}
}

func (s *Service) ToggleDb(doTurnOn bool) bool {
	var statCheck string
	// Enter SvCLI Mode
	s.Client.SendCmd(SrpCmds.SvCLI)
	switch {
	case doTurnOn:
		statCheck = "online"
		// Start Services
		s.Client.SendCmd(SrpCmds.StartDb)
		s.Client.SendCmd(SrpCmds.StartAppServ)
	case !doTurnOn:
		statCheck = "offline"
		// Stop Services
		s.Client.SendCmd(SrpCmds.StopDb)
		s.Client.SendCmd(SrpCmds.StopAppServ)
	}
	dbStat, _ := s.ShowDbServStatus()
	if contains(dbStat.JBossAppServer, statCheck) && contains(dbStat.JBossAppServerWrapper, statCheck) &&
		contains(dbStat.PgSQL, statCheck) {
		return true
	}
	return s.ToggleDb(doTurnOn)
}

// LongVacuum process used when DB and|or App Server have crashed
func (s *Service) LongVacuum() []string {
	var cmdResults []string
	// Ensure DB's are properly shutdown
	s.ToggleDb(false)
	// Exit back to "svadmin mode"
	s.Client.SendCmd("exit")
	// Switch User
	out, _ := s.Client.SendCmd(SrpCmds.SwitchUser)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	// Switch to pgsql user
	out, _ = s.Client.SendCmd(SrpCmds.SwitchUserPg)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	out, _ = s.Client.SendCmd(SrpCmds.PgSvStat)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	out, _ = s.Client.SendCmd("vacuum analyze")
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	s.Client.SendCmd(SrpCmds.CntrlD)

	out, _ = s.Client.SendCmd(SrpCmds.PgSvSub)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	out, _ = s.Client.SendCmd("vacuum analyze")
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	// Exit out of this Configuration Mode
	s.Client.SendCmd(SrpCmds.CntrlD)
	s.Client.SendCmd(SrpCmds.CntrlD)
	out, _ = s.Client.SendCmd(SrpCmds.CntrlD)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	// Toggle DB
	s.ToggleDb(true)
	return cmdResults
}

// ShortVacuum process used before a DB|App Server Crash occurs
func (s *Service) ShortVacuum() []string {
	var cmdResults []string
	out, _ := s.Client.SendCmd(SrpCmds.PgSvStatShort)
	// Verify DB is actually Operational
	if contains(out, SrpCmds.PgErr) {
		// If the DB is actually Shutdown then This Process Will Not Work
		return s.LongVacuum()
	}
	log.Println("enter pg sv stat mode")
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	s.Client.SendCmd("vacuum analyze;")
	log.Println("finished db vacuum on sv_stat")
	time.Sleep(500 * time.Millisecond)
	out, _ = s.Client.SendCmd(SrpCmds.Quit)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}

	out, _ = s.Client.SendCmd(SrpCmds.PgSvSubShort)
	// Verify DB is actually Operational
	if contains(out, SrpCmds.PgErr) {
		// If the DB is actually Shutdown then This Process Will Not Work
		return s.LongVacuum()
	}
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	log.Println("enter pg sv sub mode")
	s.Client.SendCmd("vacuum analyze;")
	log.Println("finished db vacuum on sv_sub")
	out, _ = s.Client.SendCmd(SrpCmds.Quit)
	if out != "" {
		cmdResults = append(cmdResults, out)
	}
	// Ensure the DB is Started If Necessary (shouldn't be)
	s.ToggleDb(true)
	return cmdResults
}
