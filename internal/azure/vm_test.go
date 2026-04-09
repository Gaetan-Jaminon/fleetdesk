package azure

import "testing"

func TestParseVMDetail(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
		check   func(t *testing.T, d VMDetail)
	}{
		{
			name: "full VM detail",
			input: []byte(`{
				"name": "vm-aap-dev-01",
				"resourceGroup": "RG-AAP-DEV",
				"location": "westeurope",
				"hardwareProfile": {"vmSize": "Standard_D2s_v3"},
				"storageProfile": {
					"osDisk": {"osType": "Linux", "name": "vm-aap-dev-01_OsDisk", "diskSizeGb": 64},
					"imageReference": {"offer": "RHEL", "publisher": "RedHat", "sku": "9-lvm-gen2"}
				},
				"powerState": "VM running",
				"privateIps": "10.0.1.4",
				"publicIps": "20.1.2.3",
				"id": "/subscriptions/xxx/resourceGroups/RG-AAP-DEV/providers/Microsoft.Compute/virtualMachines/vm-aap-dev-01",
				"tags": {"env": "dev", "owner": "platform"},
				"networkProfile": {
					"networkInterfaces": [{"id": "/subscriptions/xxx/resourceGroups/RG-AAP-DEV/providers/Microsoft.Network/networkInterfaces/vm-aap-dev-01-nic"}]
				},
				"timeCreated": "2025-01-15T10:30:00.000000+00:00"
			}`),
			check: func(t *testing.T, d VMDetail) {
				if d.Name != "vm-aap-dev-01" {
					t.Errorf("Name = %q", d.Name)
				}
				if d.PowerState != "running" {
					t.Errorf("PowerState = %q, want running", d.PowerState)
				}
				if d.OSDiskName != "vm-aap-dev-01_OsDisk" {
					t.Errorf("OSDiskName = %q", d.OSDiskName)
				}
				if d.OSDiskSizeGB != 64 {
					t.Errorf("OSDiskSizeGB = %d, want 64", d.OSDiskSizeGB)
				}
				if d.NICName != "vm-aap-dev-01-nic" {
					t.Errorf("NICName = %q", d.NICName)
				}
				if d.Tags["env"] != "dev" {
					t.Errorf("Tags[env] = %q, want dev", d.Tags["env"])
				}
				if d.CreatedTime != "2025-01-15T10:30:00.000000+00:00" {
					t.Errorf("CreatedTime = %q", d.CreatedTime)
				}
				if d.PublicIP != "20.1.2.3" {
					t.Errorf("PublicIP = %q", d.PublicIP)
				}
				if d.OSDisk != "RHEL 9-lvm-gen2" {
					t.Errorf("OSDisk = %q", d.OSDisk)
				}
			},
		},
		{
			name: "minimal fields",
			input: []byte(`{
				"name": "vm-min",
				"resourceGroup": "RG",
				"location": "westeurope",
				"hardwareProfile": {"vmSize": "Standard_B1s"},
				"storageProfile": {"osDisk": {"osType": "Linux"}, "imageReference": null},
				"powerState": "VM deallocated",
				"privateIps": "10.0.1.1",
				"publicIps": "",
				"id": "/sub/xxx",
				"networkProfile": {"networkInterfaces": []}
			}`),
			check: func(t *testing.T, d VMDetail) {
				if d.OSDisk != "" {
					t.Errorf("OSDisk = %q, want empty", d.OSDisk)
				}
				if d.NICName != "" {
					t.Errorf("NICName = %q, want empty", d.NICName)
				}
				if d.PowerState != "deallocated" {
					t.Errorf("PowerState = %q", d.PowerState)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   []byte(`not json`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseVMDetail(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, d)
			}
		})
	}
}

func TestSplitLast(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/a/b/c/nic-name", "nic-name"},
		{"no-slash", "no-slash"},
		{"/single", "single"},
		{"", ""},
	}
	for _, tt := range tests {
		got := splitLast(tt.input, "/")
		if got != tt.want {
			t.Errorf("splitLast(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseGraphVMs(t *testing.T) {
	input := []byte(`{
		"count": 2,
		"data": [
			{
				"name": "vm-01",
				"resourceGroup": "rg-app-dev",
				"location": "westeurope",
				"id": "/subscriptions/xxx/resourceGroups/rg-app-dev/providers/Microsoft.Compute/virtualMachines/vm-01",
				"powerState": "PowerState/running",
				"vmSize": "Standard_D2s_v3",
				"osType": "Linux",
				"offer": "RHEL",
				"sku": "9-lvm-gen2",
				"computerName": "vm-01.cnp.dev.fluxys.cloud"
			},
			{
				"name": "vm-02",
				"resourceGroup": "rg-app-dev",
				"location": "westeurope",
				"id": "/subscriptions/xxx/resourceGroups/rg-app-dev/providers/Microsoft.Compute/virtualMachines/vm-02",
				"powerState": "PowerState/deallocated",
				"vmSize": "Standard_D4s_v3",
				"osType": "Windows",
				"offer": "WindowsServer",
				"sku": "2022-datacenter-g2",
				"computerName": "VM-02"
			}
		]
	}`)
	vms, err := ParseGraphVMs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("got %d VMs, want 2", len(vms))
	}
	if vms[0].PowerState != "running" {
		t.Errorf("vms[0].PowerState = %q, want running", vms[0].PowerState)
	}
	if vms[0].Hostname != "vm-01.cnp.dev.fluxys.cloud" {
		t.Errorf("vms[0].Hostname = %q", vms[0].Hostname)
	}
	if vms[1].PowerState != "deallocated" {
		t.Errorf("vms[1].PowerState = %q, want deallocated", vms[1].PowerState)
	}
	if vms[1].OSDisk != "WindowsServer 2022-datacenter-g2" {
		t.Errorf("vms[1].OSDisk = %q", vms[1].OSDisk)
	}
}

func TestParseGraphNICs(t *testing.T) {
	input := []byte(`{
		"count": 2,
		"data": [
			{"vmId": "/subscriptions/xxx/providers/Microsoft.Compute/virtualMachines/vm-01", "privateIp": "10.0.1.4", "subnetId": "/subscriptions/xxx/resourceGroups/RG/providers/Microsoft.Network/virtualNetworks/VNET-APP-DEV/subnets/AppSubnet"},
			{"vmId": "/subscriptions/xxx/providers/Microsoft.Compute/virtualMachines/vm-02", "privateIp": "10.0.1.5", "subnetId": "/subscriptions/xxx/resourceGroups/RG/providers/Microsoft.Network/virtualNetworks/VNET-MAN-DEV/subnets/MgmtSubnet"}
		]
	}`)
	m, err := ParseGraphNICs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d entries, want 2", len(m))
	}
	info := m["/subscriptions/xxx/providers/microsoft.compute/virtualmachines/vm-01"]
	if info.PrivateIP != "10.0.1.4" {
		t.Errorf("vm-01 IP = %q", info.PrivateIP)
	}
	if info.VNet != "VNET-APP-DEV" {
		t.Errorf("vm-01 VNet = %q, want VNET-APP-DEV", info.VNet)
	}
	if info.Subnet != "AppSubnet" {
		t.Errorf("vm-01 Subnet = %q, want AppSubnet", info.Subnet)
	}
	info2 := m["/subscriptions/xxx/providers/microsoft.compute/virtualmachines/vm-02"]
	if info2.VNet != "VNET-MAN-DEV" {
		t.Errorf("vm-02 VNet = %q, want VNET-MAN-DEV", info2.VNet)
	}
}

func TestParseSubnetID(t *testing.T) {
	tests := []struct {
		input      string
		wantVNet   string
		wantSubnet string
	}{
		{"/subscriptions/xxx/resourceGroups/RG/providers/Microsoft.Network/virtualNetworks/VNET-APP/subnets/Web", "VNET-APP", "Web"},
		{"", "", ""},
		{"/no/matching/path", "", ""},
	}
	for _, tt := range tests {
		vnet, subnet := parseSubnetID(tt.input)
		if vnet != tt.wantVNet {
			t.Errorf("parseSubnetID(%q) vnet = %q, want %q", tt.input, vnet, tt.wantVNet)
		}
		if subnet != tt.wantSubnet {
			t.Errorf("parseSubnetID(%q) subnet = %q, want %q", tt.input, subnet, tt.wantSubnet)
		}
	}
}

func TestParseVMPowerStates(t *testing.T) {
	input := []byte(`{
		"count": 3,
		"data": [
			{"name": "vm-01", "powerState": "PowerState/running"},
			{"name": "vm-02", "powerState": "PowerState/starting"},
			{"name": "vm-03", "powerState": "PowerState/deallocated"}
		]
	}`)
	states, err := ParseVMPowerStates(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("got %d states, want 3", len(states))
	}
	if states["vm-01"] != "running" {
		t.Errorf("vm-01 = %q, want running", states["vm-01"])
	}
	if states["vm-02"] != "starting" {
		t.Errorf("vm-02 = %q, want starting", states["vm-02"])
	}
	if states["vm-03"] != "deallocated" {
		t.Errorf("vm-03 = %q, want deallocated", states["vm-03"])
	}
}
