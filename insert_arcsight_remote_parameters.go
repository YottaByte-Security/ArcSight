package main

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//Written by Bob Failla
//
// Description:
//		- Inputs base path from the user
//		- Inputs starting port number
//		- Determines from user whether or not safeties should be applied
//		- Finds agent.properties files by recursively transcending the base path
//		- Backs up the agent.properties files to a timestamped (RFC3339) backup file
//		- Searches agent.properties line by line
//		- Does not change existing agent.properties file.  Instead writes to agent.properties.new
//		- Removes any existing remote properties parameters, except organization parameter
//		- attains computer host name ***
//		- appends remote management parameters to the end of agent.properties.new
//		- increments the port accumulator for the next connector
//		- continues to the next connector until finished updating all connectors
//		- timestamps and logs all actions to a log file dropped into the base directory
//		- logs the ending port number


//  Status - Failed testing of moving agent.properties.bak to agent.properties |  Todos remaining

// Porting Checklist:
//    	0)   / may need to change to \.
//		0) When porting between different OS, the following lines need to be examined for path seperator:
//		0) 		attr.path += "/user/agent/agent.properties"
//		0)		log_file := base + "/remote_parameter_update.log"
//		1)	make sure the os.host supplies a FQDN
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"os"
	"fmt"
	"path/filepath"
	"regexp"
	"bufio"
	"io"
	"time"
	"log"
	"strconv"
)
///////////////////////////////////////////////////////////////////////
type fileattr struct {
	fileinfo os.FileInfo
	path     string
}
///////////////////////////////////////////////////////////////////////
var (
	base 			string
	logger			*os.File
	safeties		bool
	port			int

)
func perform_safe_operation 		() 													(safeties bool) 	{
	fmt.Println("Operating without safties in place means changes will be made.")
	fmt.Println("Operating with safties means that no changes will be made; but log entries of what")
	fmt.Println("would have been done will be logged?")
	fmt.Printf("Safties?:")
	safeties = true
	s_input := "Yes"
	done := false
	for ; done != true; {
		fmt.Scanf("%s\n", &s_input)
		switch {
		case s_input =="Y"|| s_input == "y" || s_input == "Yes" || s_input == "YES" || s_input == "yes" : {
			safeties = true
			done = true
		} // end of affirmative case
		case s_input == "N" || s_input == "n" || s_input == "No"|| s_input == "NO" || s_input == "no":{
			safeties = false
			done = true
		} // end of negative case
		} // end of switch
	} // end of for statement
	return safeties
}
func get_base_dir 					() 													(base string)		{
	fmt.Println("Please enter the base directory for the ArcSight Connectors:")
	fmt.Scanf("%s\n", &base)
	return base
}
func set_up_logging 				(base string) 										(logger *os.File) 	{
	log_file := base + "/remote_parameter_update.log"
	logger, err := os.OpenFile(log_file, os.O_RDWR |os.O_CREATE |os.O_APPEND, 0666)
	if err != nil {
		panic(err)
		fmt.Println("unable to open log")
	}
	log.SetOutput(logger)
	return logger
}
func get_starting_port 				()													(port int) 			{
	fmt.Printf("Please enter the starting port number for this host:")
	fmt.Scanf("%d\n", &port)
	fmt.Println()
	return port
}
func safe_copy				 		(source, destination string)  						() 					{
	var (
		original 	*os.File
		err			error
	)
	t := time.Now()
	original, err = os.Open(source)
	if err != nil {
		log.Println(t.Format(time.RFC3339) + " Fatal: Failed to open " + source + " for backup")
		panic(err)
	}
	dest, err := os.Create(destination)
	if err != nil {
		log.Println(t.Format(time.RFC3339) + " Fatal: Failed to create backup destination")
		panic(err)
	}
	io.Copy(dest, original)
	if err != nil {
		log.Println(t.Format(time.RFC3339) + "Fatal:  Failed to make the backup copies")
		panic(err)
	} else {
		dest.Close()
		original.Close()
		log.Println(t.Format(time.RFC3339) + " Success:  Copy Successful" + source +" to " + destination)
	}
}
func edit_properties_file 			(properties_file string, safeties bool, port int) 	(new_port int)		{
	var (
		output_file	*os.File
		remote_management_match, remote_user_match, remote_management_organization_match, remote_match bool
	)
	t := time.Now()
	original, _ := os.Open(properties_file)
	input_file 	:= bufio.NewScanner(original)
	if safeties != true {output_file, _ = os.Create(properties_file+ ".new")}
	////////////////////////// ****************** need error handling for opening the file for writing *****************
		for input_file.Scan() {
			line := input_file.Text()
			remote_management_match, _ = 						regexp.MatchString("^remote.management", line)
			remote_user_match, _ = 								regexp.MatchString("^remote.user", line)
			remote_management_organization_match, _ = 			regexp.MatchString("^remote.management.ssl.organizational.unit", line)
			remote_match, _ = 									regexp.MatchString("^remote", line)
			switch {
				case remote_management_organization_match : {
					if safeties == true {
						log.Println(line)} else {
						output_file.WriteString(line + "\n")}}
				case remote_user_match: {break} //Do not write the line - effective delete
				case remote_management_match : {break}  // Do not write the line - effective delete
				case remote_match: {break}  //Do not write the line - effective delete
				default: {
					if safeties != true {output_file.WriteString(line + "\n")} else { log.Println(line)}}
			}
		}
	host_name, _ := os.Hostname()
	host_param := ("remote.management.host=" + host_name)
	port_param := ("remote.management.port=" + strconv.Itoa(port))
	if safeties != true {
		output_file.WriteString("remote.management.enabled=true\n")
		output_file.WriteString(host_param + "\n")
		output_file.WriteString(port_param + "\n")
		output_file.WriteString("remote.user=lemon\n")
		log.Println(t.Format(time.RFC3339) + " Success:  Remote parameters added")
	} else {
		log.Println("remote.management.enabled=True")
		log.Println(host_param)
		log.Println(port_param)
		log.Println("remote.user=lemon")
	}
	port += 1
	return port
}
func backup_and_edit				(path string, f os.FileInfo, _ error) 				(error) 			{
	//
	// Find agent.properties files, backup and modify if necessary
	//
	var (
		attr fileattr
	)
	t := time.Now()
	attr.fileinfo = f
	attr.path = path
	if attr.fileinfo.IsDir() {
		attr.path += "/user/agent/agent.properties"
		backup_file := path + "/user/agent/"+t.Format(time.RFC3339)+"agent.properties.bak"
		if _, err := os.Stat(attr.path); os.IsNotExist(err) {
			// Function written in the negative - sorry os.IsExist(err) doesn't give same result
		} else {
			log.Println("Info: Found ", attr.path)
			if safeties == true {
				log.Println(t.Format(time.RFC3339) + " Warning:  Safeties on.  Copy " + attr.path + " to " + backup_file + " not attempted.")
			} else {
				safe_copy(attr.path, backup_file)
			}
			new_port := edit_properties_file(attr.path, safeties, port)
			port = new_port
			if safeties == true {
				log.Println(t.Format(time.RFC3339) + " Warning:  Safeties on.  Copy" + backup_file + " to " + attr.path + " not attempted.")
			} else {
				safe_copy((attr.path +".new"), attr.path)
				os.Remove(attr.path+".new")
				log.Println(t.Format(time.RFC3339) + " Success: Removed " + attr.path + ".new.")

			}
		}
	}
	return nil
}
func main () {

	t := time.Now()
	base 		=	get_base_dir ()
	logger 		= 	set_up_logging(base)
	log.Println(t.Format(time.RFC3339) + " Info:  Base directory defined as " + base)
	safeties 	= 	perform_safe_operation()
	log.Printf(t.Format(time.RFC3339) + " Warning:  Safties set to %t\n", safeties)
	port 		= 	get_starting_port()
	log.Printf(t.Format(time.RFC3339) + " Info:  Starting Port number set to: %d\n", port)
	filepath.Walk(base, backup_and_edit)
	log.Printf(t.Format(time.RFC3339) + " Info:  Ending port is %d", port-1)

} // Program end
