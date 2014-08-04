package main
import (
	"dmzhang/catkeeper/libvirt"
	"dmzhang/catkeeper/vminstall"
	"dmzhang/catkeeper/utils"
	"fmt"
	"flag"
	"os/exec"
	"time"
)

func usage() {
	/*
	fmt.Println("USAGE:")
	fmt.Println("vminstallbin -name=<NAME> -repo=<REPOSITORY URL> -host=<PHYSICAL HOST IPADDRESS> -size= <num>(UNIT G), -autoyast=<HTTP://XMLLOCATION>")
	*/
	fmt.Println("\nEXAMPLE:")
	fmt.Println("vminstallbin -host=147.2.207.234 -name=my_test -repo=http://147.2.207.240/repo/SLP/sles10/")
	fmt.Println("vminstallbin -host=147.2.207.234 -name=my_test -repo=http://147.2.207.240/repo/SLP/sles10/ --autoyast=http://147.2.207.233/ay/autoinst.xml")

}

func main() {

	var (
		hostPtr = flag.String("host", "", "remote host IP address")
		repoPtr = flag.String("repo", "", "installation repository")
		autoyastPtr = flag.String("autoyast", "", "location of autoyast xml")
		imageSizePtr = flag.Uint64("size", 10, "image size (G)")
		namePtr = flag.String("name", "", "name of the Virtual Machine")
	)

	flag.Parse()

	var remoteURL string
	if *hostPtr == "" {
		remoteURL = "qemu+ssh:///system"
	} else {
		remoteURL = "qemu+ssh://root@" + *hostPtr+ "/system"
	}

	if *repoPtr == "" {
		fmt.Println("MISSING repo")
		usage()
		return
	}
	repo := *repoPtr

	if *namePtr == "" {
		fmt.Println("MISSING name")
		usage()
		return
	}
	name := *namePtr

	if *autoyastPtr == "" {
		fmt.Println("You did not have autoyast.xml")
	}
	autoinst := *autoyastPtr

	// GB > Byte
	imageSize := *imageSizePtr << 30


	fmt.Printf("Install From :%s \n" , remoteURL)
	fmt.Printf("Name         :%s \n" , name)
	fmt.Printf("Disk Size    :%dG\n", *imageSizePtr)
	fmt.Printf("Repository   :%s \n" , repo)
	fmt.Printf("AutoYast     :%s \n" , autoinst)


	// create remote pool
	fmt.Printf("Creating connection to %s\n", *hostPtr)
	conn, err := libvirt.NewVirConnection(remoteURL)
	if (err != nil) {
		fmt.Println(err)
		return
	}
	defer conn.CloseConnection()

	ch := make(chan string)


	go vminstall.VmInstall(conn, name, repo, autoinst, uint64(imageSize), ch)

	var quiltChan = make(chan bool)
	for m := range ch {
		if m == vminstall.VMINSTALL_SUCCESS {
			break
		}
		fmt.Println(m)
	}
	startVNCviewer(conn, name, *hostPtr, quiltChan)
}

type listener struct {
	quitchan chan bool
}
func (this *listener) FreeHandle() {
}

func (this *listener) EventHandle(conn libvirt.VirConnection, domain libvirt.VirDomain, event int, detail int) {
	fmt.Println(event)
	if event ==  libvirt.VIR_DOMAIN_EVENT_STOPPED {
		//to restart the domain
		domain.Create()
		this.quitchan <- true
	}
}

func startVNCviewer(conn libvirt.VirConnection, name string, hostIPAddress string, quiltchan chan bool) {
	fmt.Println("would bring up vncviewer...")
	var domain libvirt.VirDomain
	domain ,err := conn.LookupByName(name)
	if err != nil {
		fmt.Println("FAIL: find running domain to start vncviewer")
		return
	}
	defer domain.Free()

	xmlData, _ := domain.GetXMLDesc()
	v := utils.ParseDomainXML(xmlData)

	/* to get VNC port */
	var vncPort string
	if (v.Devices.Graphics.VNCPort == "-1") {
		fmt.Println("FAIL:Can not get vnc port")
		return
	}

	vncPort =  v.Devices.Graphics.VNCPort
	fmt.Println("Running reboot listener")

	go func(){
		libvirt.EventRegisterDefaultImpl()
		// EventRunDefaultImpl has to be run before register. or no events caught,
		// I do not know why
		go func(){
			for {
			libvirt.EventRunDefaultImpl()
		}}()

		l := listener{quiltchan}
	        libvirt.ConnectDomainEventRegister(conn, domain,  &l)
		for {
			time.Sleep(1)
		}
	}()

	fmt.Println("RUNNING: vncviewer " + hostIPAddress + ":" + vncPort)
	go func() {
		cmd := exec.Command("vncviewer", hostIPAddress + ":" + vncPort)
		//Run will block
		err = cmd.Run()
		if err != nil {
			fmt.Println("FAIL:can not start vncviewer")
			fmt.Println(err)
			return
		}
		quiltchan <- true
	}()

	//either get reboot event or user quit the vncviewer gui, this application will quit
	<-quiltchan
}
