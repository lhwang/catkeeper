package libvirt


/*
#include <libvirt/libvirt.h>

int libvirt_lifecycle_eventcallback_cgo(virConnectPtr c, virDomainPtr d, int event, int detail, void * data) {
	LifeCycleCallBack(c, d, event, detail, data);
}

int libvirt_generic_eventcallback_cgo(virConnectPtr c, virDomainPtr d,  void * opaque) {
	GenericCallBack(c, d, opaque);
}

void libvirt_virfreecalback_cgo(void *opaque){
	VirFreeCallback(opaque);
}
*/
import "C"
