package commands

import (
	"database/sql"
	"strings"
)

// EnhanceCommandDescriptions adds searchable keywords to common commands
// This is simpler than complex intent detection - let descriptions do the work
func EnhanceCommandDescriptions(db *sql.DB) error {
	enhancements := map[string]string{
		// Network/Port commands
		"ss":      "sockets ports listening connections tcp udp network check show port",
		"netstat": "network connections ports listening tcp udp sockets statistics",
		"lsof":    "open files ports processes sockets connections network check",
		"nmap":    "network scan ports security reconnaissance discover hosts",
		"nc":      "netcat network connection tcp udp port test listen connect socket",

		// Process commands
		"ps":    "processes running pid list show status cpu memory",
		"top":   "processes monitor watch cpu memory running performance",
		"htop":  "processes interactive monitor cpu memory performance system",
		"kill":  "terminate stop process pid end signal force",
		"pkill": "kill processes name pattern terminate stop",

		// File operations
		"cp":     "copy files directories duplicate backup",
		"mv":     "move rename files directories relocate",
		"rm":     "remove delete files directories unlink erase",
		"find":   "search locate files directories name pattern",
		"locate": "search find files quickly database index",

		// File system
		"df":     "disk space usage filesystem free available storage",
		"du":     "disk usage size files directories space occupied",
		"mount":  "filesystem attach disk partition usb drive storage",
		"umount": "unmount detach filesystem disk remove eject",

		// Archives
		"tar":   "archive compress extract backup zip files bundle",
		"gzip":  "compress decompress zip files reduce size",
		"unzip": "extract decompress archive zip files",

		// Text/Search
		"grep": "search pattern text files find match filter",
		"sed":  "edit text stream replace pattern transform",
		"awk":  "text processing columns fields data manipulation",
		"cat":  "display show read concatenate files output print",
		"less": "view read pager scroll files text display",
		"tail": "view last lines follow monitor watch log files",
		"head": "view first lines beginning start files display",

		// System info
		"uname":    "system information kernel version platform os",
		"hostname": "computer name host system network identifier",
		"uptime":   "system running time load average boot status",
		"free":     "memory ram usage available swap statistics",

		// Permissions
		"chmod": "change permissions mode access rights files",
		"chown": "change owner ownership user group files",
		"sudo":  "superuser root admin privileges execute command",

		// Network tools
		"ping":  "test network connectivity reachable host check connection",
		"curl":  "download http request url api web fetch",
		"wget":  "download files http ftp url fetch retrieve",
		"ssh":   "remote login secure shell connect server terminal",
		"scp":   "secure copy files remote ssh transfer upload",
		"rsync": "sync synchronize copy files backup mirror remote",
	}

	// Update descriptions in database
	for cmd, keywords := range enhancements {
		// Get existing description
		var existingDesc string
		err := db.QueryRow("SELECT description FROM commands WHERE name = ?", cmd).Scan(&existingDesc)
		if err != nil {
			continue // Command not found, skip
		}

		// Append keywords if not already present
		enhanced := existingDesc
		for _, keyword := range strings.Fields(keywords) {
			if !strings.Contains(strings.ToLower(enhanced), keyword) {
				enhanced += " " + keyword
			}
		}

		// Update in database
		_, err = db.Exec("UPDATE commands SET description = ? WHERE name = ?", enhanced, cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCommandKeywords returns additional search keywords for a command
// This is used during indexing to boost discoverability
func GetCommandKeywords(commandName string) []string {
	// Common patterns and synonyms
	keywords := map[string][]string{
		"ss":      {"port", "socket", "connection", "listening", "tcp", "udp"},
		"netstat": {"port", "socket", "connection", "network"},
		"lsof":    {"port", "socket", "open", "file", "process"},
		"cp":      {"copy", "duplicate", "backup"},
		"mv":      {"move", "rename"},
		"rm":      {"delete", "remove", "erase"},
		"ps":      {"process", "pid", "running"},
		"grep":    {"search", "find", "match", "filter"},
		"find":    {"search", "locate"},
	}

	return keywords[commandName]
}
