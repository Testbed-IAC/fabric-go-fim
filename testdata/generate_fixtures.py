#!/usr/bin/env python3
"""
generate_fixtures.py — Generate GraphML fixtures from Python FIM reference implementation.

Usage:
    python3 testdata/generate_fixtures.py

Installs fabric-fim via pip if not present, then writes one GraphML file per
topology pattern to testdata/fixtures/<pattern_name>.graphml.

These files are the ground truth for the Go golden tests in pkg/topology/golden_test.go.
"""

import os
import sys
import subprocess
import importlib
import importlib.util
import traceback
from pathlib import Path

FIXTURES_DIR = Path(__file__).parent / "fixtures"
TOTAL_FIXTURES = 28  # 15 patterns + 13 catalog models

# ---------------------------------------------------------------------------
# Dependency bootstrap
# ---------------------------------------------------------------------------

def _ensure_fim():
    """Install fabric-fim if not importable."""
    if importlib.util.find_spec("fim") is not None:
        return
    print("fabric-fim not found — installing via pip ...", flush=True)
    subprocess.check_call([sys.executable, "-m", "pip", "install", "fabric-fim"],
                          stdout=subprocess.DEVNULL)
    print("fabric-fim installed.", flush=True)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _write(name: str, graphml_str: str):
    """Write a GraphML fixture, normalising line endings."""
    FIXTURES_DIR.mkdir(parents=True, exist_ok=True)
    path = FIXTURES_DIR / f"{name}.graphml"
    path.write_text(graphml_str, encoding="utf-8")
    print(f"  wrote {path.relative_to(Path(__file__).parent.parent)}")


def _serialize(t) -> str:
    """Serialize an ExperimentTopology to a GraphML string."""
    # fabric-fim serialize() writes to file and returns the string.
    # Call without file_name to get string-only (if supported), else use a temp path.
    try:
        result = t.serialize()
        if result:
            return result
    except Exception:
        pass
    # Fallback: write to a temp file and read back.
    import tempfile
    with tempfile.NamedTemporaryFile(suffix=".graphml", delete=False) as f:
        tmp = f.name
    try:
        t.serialize(file_name=tmp)
        return Path(tmp).read_text(encoding="utf-8")
    finally:
        try:
            os.unlink(tmp)
        except OSError:
            pass


def _iface(component):
    """Return the first interface (ConnectionPoint) of a component as an object."""
    ifaces = list(component.interface_list)
    if not ifaces:
        raise RuntimeError(f"component {component.name!r} has no interfaces")
    iface = ifaces[0]
    # interface_list may yield dict-like objects or proper interface objects.
    # If it's a dict, fall back to topology.get_interface_by_name.
    if isinstance(iface, dict):
        raise RuntimeError(
            f"interface_list returned dict; cannot use as interface object: {iface}"
        )
    return iface


# ---------------------------------------------------------------------------
# Topology builders — one per pattern
# ---------------------------------------------------------------------------

def build_bare_vm(ET, Capacities, **_):
    t = ET()
    t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    return _serialize(t)


def build_vm_shared_nic(ET, Capacities, ComponentType, **_):
    t = ET()
    vm = t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    vm.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    return _serialize(t)


def build_vm_smart_nic(ET, Capacities, ComponentType, **_):
    t = ET()
    vm = t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    vm.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic")
    return _serialize(t)


def build_vm_gpu(ET, Capacities, ComponentType, **_):
    t = ET()
    vm = t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    vm.add_component(ctype=ComponentType.GPU, model="RTX6000", name="gpu1")
    return _serialize(t)


def build_vm_nvme(ET, Capacities, ComponentType, **_):
    t = ET()
    vm = t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    vm.add_component(ctype=ComponentType.NVME, model="P4510", name="ssd")
    return _serialize(t)


def build_vm_subinterface(ET, Capacities, ComponentType, Labels, **_):
    t = ET()
    vm = t.add_node(
        name="vm1",
        site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9",
        image_type="qcow2",
    )
    snic = vm.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic")
    parent_iface = _iface(snic)
    parent_iface.add_child_interface(
        name="sub100",
        labels=Labels(vlan="100"),
    )
    return _serialize(t)


def build_lan_l2bridge(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    c2 = vm2.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    t.add_network_service(
        name="lan1",
        nstype=ServiceType.L2Bridge,
        interfaces=[_iface(c1), _iface(c2)],
    )
    return _serialize(t)


def build_lan_l2sts(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="UKY",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    c2 = vm2.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    t.add_network_service(
        name="lan1",
        nstype=ServiceType.L2STS,
        interfaces=[_iface(c1), _iface(c2)],
    )
    return _serialize(t)


def build_l2ptp(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="UKY",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic1")
    c2 = vm2.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic1")
    # Explicitly select p1 for both VMs — interface_list order is non-deterministic
    # in Python FIM, so use a named lookup to produce a stable fixture.
    iface1 = next(i for i in c1.interface_list if i.name == "snic1-p1")
    iface2 = next(i for i in c2.interface_list if i.name == "snic1-p1")
    t.add_network_service(
        name="ptp1",
        nstype=ServiceType.L2PTP,
        interfaces=[iface1, iface2],
    )
    return _serialize(t)


def build_fabnetv4(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    c2 = vm2.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    t.add_network_service(
        name="v4net",
        nstype=ServiceType.FABNetv4,
        interfaces=[_iface(c1), _iface(c2)],
    )
    return _serialize(t)


def build_fabnetv6(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    c2 = vm2.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    t.add_network_service(
        name="v6net",
        nstype=ServiceType.FABNetv6,
        interfaces=[_iface(c1), _iface(c2)],
    )
    return _serialize(t)


def build_facility_port(ET, Capacities, Labels, **_):
    t = ET()
    t.add_facility(
        name="ESnet-DTN",
        site="RENC",
        nslabels=Labels(vlan="100"),
        capacities=Capacities(bw=10),
    )
    return _serialize(t)


def build_switch_node(ET, Capacities, **_):
    t = ET()
    t.add_switch(
        name="sw1",
        site="RENC",
        nports=4,
    )
    return _serialize(t)


def build_port_mirror(ET, Capacities, ComponentType, ServiceType, MirrorDirection, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    c2 = vm2.add_component(ctype=ComponentType.SharedNIC, model="ConnectX-6", name="nic1")
    iface0 = _iface(c1)
    iface1 = _iface(c2)
    t.add_network_service(
        name="lan1",
        nstype=ServiceType.L2Bridge,
        interfaces=[iface0],
    )
    t.add_port_mirror_service(
        name="pm1",
        from_interface_name=iface0.name,
        to_interface=iface1,
        from_interface_vlan="100",
        direction=MirrorDirection.Both,
    )
    return _serialize(t)


def build_explicit_link(ET, Capacities, ComponentType, ServiceType, **_):
    t = ET()
    vm1 = t.add_node(
        name="vm1", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    vm2 = t.add_node(
        name="vm2", site="RENC",
        capacities=Capacities(core=2, ram=8, disk=10),
        image_ref="default_rocky_9", image_type="qcow2",
    )
    c1 = vm1.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic1")
    c2 = vm2.add_component(ctype=ComponentType.SmartNIC, model="ConnectX-6", name="snic1")
    # Explicitly select p1 for both VMs — interface_list order is non-deterministic.
    iface1 = next(i for i in c1.interface_list if i.name == "snic1-p1")
    iface2 = next(i for i in c2.interface_list if i.name == "snic1-p1")
    # Python FIM uses add_network_service(nstype=L2Path) — creates NetworkService
    # + service ports + patch links, not a bare Link node.
    t.add_network_service(
        name="phys-link",
        nstype=ServiceType.L2Path,
        interfaces=[iface1, iface2],
    )
    return _serialize(t)


# ---------------------------------------------------------------------------
# Per-catalog-model fixtures
# ---------------------------------------------------------------------------

# Each entry: (ComponentType attr name, model string, fixture suffix)
CATALOG_MODELS = [
    ("SharedNIC",  "ConnectX-6",         "catalog_shared_nic_connectx6"),
    ("SmartNIC",   "ConnectX-5",         "catalog_smart_nic_connectx5"),
    ("SmartNIC",   "ConnectX-6",         "catalog_smart_nic_connectx6"),
    ("SmartNIC",   "ConnectX-7-100",     "catalog_smart_nic_connectx7_100"),
    ("SmartNIC",   "ConnectX-7-400",     "catalog_smart_nic_connectx7_400"),
    ("SmartNIC",   "BlueField-2-ConnectX-6", "catalog_smart_nic_bf2"),
    ("GPU",        "Tesla T4",           "catalog_gpu_teslaT4"),
    ("GPU",        "RTX6000",            "catalog_gpu_rtx6000"),
    ("GPU",        "A30",                "catalog_gpu_a30"),
    ("GPU",        "A40",                "catalog_gpu_a40"),
    ("NVME",       "P4510",              "catalog_nvme_p4510"),
    ("FPGA",       "Xilinx-U280",        "catalog_fpga_u280"),
    ("FPGA",       "Xilinx-SN1022",      "catalog_fpga_sn1022"),
]


def build_catalog_fixtures(ET, Capacities, ComponentType, **_):
    results = {}
    for ctype_attr, model, fixture_name in CATALOG_MODELS:
        try:
            ctype_val = getattr(ComponentType, ctype_attr, None)
            if ctype_val is None:
                print(f"    WARNING: ComponentType has no attribute {ctype_attr!r} — skipping {fixture_name}")
                continue
            t = ET()
            vm = t.add_node(
                name="vm1",
                site="RENC",
                capacities=Capacities(core=2, ram=8, disk=10),
                image_ref="default_rocky_9",
                image_type="qcow2",
            )
            vm.add_component(ctype=ctype_val, model=model, name="dev")
            results[fixture_name] = _serialize(t)
        except Exception as exc:
            print(f"    WARNING: could not build fixture {fixture_name!r}: {exc}")
    return results


# ---------------------------------------------------------------------------
# FIM import
# ---------------------------------------------------------------------------

def _import_fim():
    """Import all FIM symbols and return a kwargs dict for builders."""
    from fim.user.topology import ExperimentTopology
    from fim.slivers.capacities_labels import Capacities, Labels

    # ComponentType (not ComponentModelType) — fabric-fim 1.9.x
    from fim.slivers.network_node import ComponentType

    # ServiceType
    ServiceType = None
    for module_path in (
        "fim.slivers.network_service",
        "fim.user.topology",
    ):
        try:
            mod = importlib.import_module(module_path)
            st = getattr(mod, "ServiceType", None)
            if st is not None:
                ServiceType = st
                break
        except ImportError:
            pass
    if ServiceType is None:
        raise ImportError("Could not import ServiceType from fabric-fim")

    # MirrorDirection
    MirrorDirection = None
    for module_path in (
        "fim.slivers.network_service",
        "fim.user.topology",
    ):
        try:
            mod = importlib.import_module(module_path)
            md = getattr(mod, "MirrorDirection", None)
            if md is not None:
                MirrorDirection = md
                break
        except ImportError:
            pass
    if MirrorDirection is None:
        # Provide a shim with the value we need
        class _MD:
            Both = "Both"
            Rx = "Rx"
            Tx = "Tx"
        MirrorDirection = _MD
        print("  WARNING: could not import MirrorDirection — using shim")

    return dict(
        ET=ExperimentTopology,
        Capacities=Capacities,
        Labels=Labels,
        ComponentType=ComponentType,
        ServiceType=ServiceType,
        MirrorDirection=MirrorDirection,
    )


# ---------------------------------------------------------------------------
# Pattern list
# ---------------------------------------------------------------------------

PATTERNS = [
    ("bare_vm",         build_bare_vm),
    ("vm_shared_nic",   build_vm_shared_nic),
    ("vm_smart_nic",    build_vm_smart_nic),
    ("vm_gpu",          build_vm_gpu),
    ("vm_nvme",         build_vm_nvme),
    ("vm_subinterface", build_vm_subinterface),
    ("lan_l2bridge",    build_lan_l2bridge),
    ("lan_l2sts",       build_lan_l2sts),
    ("l2ptp",           build_l2ptp),
    ("fabnetv4",        build_fabnetv4),
    ("fabnetv6",        build_fabnetv6),
    ("facility_port",   build_facility_port),
    ("switch_node",     build_switch_node),
    ("port_mirror",     build_port_mirror),
    ("explicit_link",   build_explicit_link),
]


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    _ensure_fim()

    print("Importing fabric-fim ...", flush=True)
    try:
        fim_kwargs = _import_fim()
    except Exception:
        traceback.print_exc()
        print(
            "\nERROR: Could not import fabric-fim symbols. Install with:\n"
            "  pip install fabric-fim\n",
            file=sys.stderr,
        )
        sys.exit(1)

    FIXTURES_DIR.mkdir(parents=True, exist_ok=True)
    generated = 0
    failed = 0

    print(f"\nGenerating {len(PATTERNS)} topology pattern fixtures ...", flush=True)
    for name, builder in PATTERNS:
        try:
            graphml_str = builder(**fim_kwargs)
            if not graphml_str:
                raise ValueError("builder returned empty string")
            _write(name, graphml_str)
            generated += 1
        except Exception as exc:
            print(f"  FAILED {name}: {exc}")
            traceback.print_exc()
            failed += 1

    print(f"\nGenerating {len(CATALOG_MODELS)} catalog model fixtures ...", flush=True)
    try:
        catalog_fixtures = build_catalog_fixtures(**fim_kwargs)
        for fixture_name, graphml_str in catalog_fixtures.items():
            _write(fixture_name, graphml_str)
            generated += 1
        catalog_failed = len(CATALOG_MODELS) - len(catalog_fixtures)
        failed += catalog_failed
    except Exception as exc:
        print(f"  FAILED catalog fixtures: {exc}")
        traceback.print_exc()
        failed += len(CATALOG_MODELS)

    total = generated + failed
    print(f"\nSummary: {generated}/{total} fixtures generated successfully")
    print(f"Output directory: {FIXTURES_DIR.resolve()}")

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
