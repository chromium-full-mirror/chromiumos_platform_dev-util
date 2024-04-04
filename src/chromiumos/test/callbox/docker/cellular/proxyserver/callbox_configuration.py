# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Configuration managers for CrOS callboxes"""

from enum import Enum
from multiprocessing.pool import ThreadPool

from acts import logger
from acts.controllers import handover_simulator as hs
from acts.controllers.cellular_lib.LteSimulation import (
    IPAddressType as _ACTSIPType,
)
from acts.controllers.rohdeschwarz_lib import cmw500
from acts.controllers.rohdeschwarz_lib import cmw500_cellular_simulator as cmw
from acts.controllers.rohdeschwarz_lib import cmx500_cellular_simulator as cmx
from acts.controllers.rohdeschwarz_lib import cmx500_events
from acts.controllers.rohdeschwarz_lib import cmx500_tx_measurement as cmx_tx
from acts.controllers.rohdeschwarz_lib.cmw500_handover_simulator import (
    Cmw500HandoverSimulator,
)
from acts.controllers.rohdeschwarz_lib.cmx500_handover_simulator import (
    Cmx500HandoverSimulator,
)
from cellular.callbox_utils.cmw500_iperf_measurement import (
    Cmw500IperfMeasurement,
)
from cellular.simulation_utils import CrOSLteSimulation


cmw500.STATE_CHANGE_TIMEOUT = 45


class CellularTechnology(Enum):
    """Supported cellular technologies."""

    LTE = "LTE"
    WCDMA = "WCDMA"
    NR5G_NSA = "NR5G_NSA"
    NR5G_SA = "NR5G_SA"


class IPAddressType(Enum):
    """Supported IP Address types."""

    IPV4 = "IPV4"
    IPV6 = "IPV6"
    IPV4V6 = "IPV4V6"


# Maps CrOS IP Types to ACTS IP types
_IP_TYPE_MAP = {
    IPAddressType.IPV4: _ACTSIPType.IPV4,
    IPAddressType.IPV6: _ACTSIPType.IPV6,
    IPAddressType.IPV4V6: _ACTSIPType.IPV4V6,
}

# Maps CrOS CellularTechnology to ACTS handover CellularTechnologies
_HANDOVER_MAP = {
    CellularTechnology.LTE: hs.CellularTechnology.LTE,
    CellularTechnology.WCDMA: hs.CellularTechnology.WCDMA,
    CellularTechnology.NR5G_NSA: hs.CellularTechnology.NR5G_NSA,
    CellularTechnology.NR5G_SA: hs.CellularTechnology.NR5G_SA,
}


class CallboxConfiguration(object):
    """Base configuration for cellular callboxes."""

    def __init__(self):
        self.host = None
        self.port = None
        self.dut = None
        self.simulator = None
        self.simulation = None
        self.parameters = None
        self.iperf = None
        self.tx_measurement = None
        self.technology = None
        self.handover = None

    def nr5g_sa_handover(self, band, channel, bw):
        """Performs a handover to 5G SA.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
            bw: the downlink bandwidth of the handover destination
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.nr5g_sa_handover(
            band, channel, bw, _HANDOVER_MAP[self.technology]
        )
        self._set_technology(CellularTechnology.NR5G_SA)

    def nr5g_nsa_handover(
        self,
        band,
        channel,
        bandwidth,
        secondary_band,
        secondary_channel,
        secondary_bandwidth,
    ):
        """Performs a handover to 5G NSA.

        Args:
            band: the band of the handover destination primary cell.
            channel: the downlink channel of the handover destination primary
                cell.
            bandwidth: the downlink bandwidth of the handover destination primary
                cell.
            secondary_band: the band of the handover destination secondary cell.
            secondary_channel: the downlink channel of the handover destination
                secondary cell.
            secondary_bandwidth: the downlink bandwidth of the handover
                destination secondary cell.
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.nr5g_nsa_handover(
            band,
            channel,
            bandwidth,
            secondary_band,
            secondary_channel,
            secondary_bandwidth,
            _HANDOVER_MAP[self.technology],
        )
        self._set_technology(CellularTechnology.NR5G_NSA)

    def lte_handover(self, band, channel, bw):
        """Performs a handover to LTE.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
            bw: the downlink bandwidth of the handover destination
        """
        raise NotImplementedError()

    def wcdma_handover(self, band, channel):
        """Performs a handover to WCDMA.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
        """
        raise NotImplementedError()

    def configure(self, parameters):
        """Configures the callbox with the provided parameters.

        Args:
            parameters: the callbox configuration dictionary.
        """
        raise NotImplementedError()

    def configure_network(self, apn, mtu, ip_type):
        """Configures the callbox network with the provided parameters.

        Args:
            apn: the callbox network access point name.
            mtu: the callbox network maximum transmission unit.
            ip_type: the callbox network IP address type.
        """
        raise NotImplementedError()

    def start(self):
        """Starts a simulation on the callbox."""
        self.simulation.start()

    def close(self):
        """Releases any resources held open by the controller."""
        self.simulator.destroy()

    def require_handover(self):
        """Verifies that handover simulations are available for the current callbox configuration.

        Raises:
            ValueError: If the feature is not supported for the current callbox configuration.
        """
        if self.handover is None:
            raise ValueError(
                f"Handovers are supported for RAT: {self.technology} in the current callbox configuration"
            )

    def require_simulation(self):
        """Verifies that CellularSimulator controls are available for the current callbox configuration.

        Raises:
            ValueError: If the feature is not supported for the current callbox configuration.
        """
        if self.simulation is None or self.simulator is None:
            raise ValueError(
                f"Action not supported for RAT: {self.technology} in the current callbox configuration"
            )

    def require_iperf(self):
        """Verifies that Iperf measurements are available for the current callbox configuration.

        Raises:
            ValueError: If the feature is not supported for the current callbox configuration.
        """
        if self.iperf is None:
            raise ValueError(
                f"Iperf not supported for RAT: {self.technology} in the current callbox configuration"
            )

    def require_tx_measurement(self):
        """Verifies that tx measurements are available for the current callbox configuration.

        Raises:
            ValueError: If the feature is not supported for the current callbox configuration.
        """
        if self.tx_measurement is None:
            raise ValueError(
                f"Tx measurement not supported for RAT: {self.technology} in the current callbox configuration"
            )


class CMW500Configuration(CallboxConfiguration):
    """Configuration for CMW500 callbox controller."""

    def __init__(self, dut, host, port, technology):
        """Initializes CMW500 controller configuration.

        Args:
            dut: a device handler implementing BaseCellularDut.
            host: the ip/host address of the callbox.
            port: the port number for the callbox controller commands.
            technology: the RAT technology.
        """
        super().__init__()

        self._lte_simulation = None
        self.dut = dut
        self.host = host
        self.port = port
        self.logger = logger.create_logger()
        self.technology = technology
        self.simulator = cmw.CMW500CellularSimulator(host, port)
        self.handover = Cmw500HandoverSimulator(self.simulator.cmw)
        self._set_technology(technology)

    def lte_handover(self, band, channel, bw):
        """Performs a handover to LTE.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
            bw: the downlink bandwidth of the handover destination
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.lte_handover(
            band, channel, bw, _HANDOVER_MAP[self.technology]
        )
        self._set_technology(CellularTechnology.LTE)

    def wcdma_handover(self, band, channel):
        """Performs a handover to WCDMA.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.wcdma_handover(
            band, channel, _HANDOVER_MAP[self.technology]
        )
        self._set_technology(CellularTechnology.WCDMA)

    def configure(self, parameters):
        """Configures the callbox with the provided parameters.

        Args:
            parameters: the callbox configuration dictionary.
        """
        self.simulation.stop()
        self.simulation.configure(parameters)
        self.simulator.wait_until_quiet()
        # reset network options to default on full reconfiguration.
        self._configure_network("internet", 1500, IPAddressType.IPV4)
        self.simulation.setup_simulator()

    def configure_network(self, apn, mtu, ip_type):
        """Configures the callbox network with the provided parameters.

        Args:
            apn: the callbox network access point name.
            mtu: the callbox network maximum transmission unit.
            ip_type: the callbox network IP address type.
        """
        # need to be detached to configure network settings
        self.simulator.stop()
        self._configure_network(apn, mtu, ip_type)
        self.simulation.setup_simulator()

    def _configure_network(self, apn, mtu, ip_type):
        self.logger.info(
            f"Configuring network: apn {apn}, mtu: {mtu}, ip_type: {ip_type}"
        )
        self.simulator.set_apn(apn)
        self.simulator.set_ip_type(_IP_TYPE_MAP[ip_type])
        self.simulator.set_mtu(mtu)
        self.simulator.wait_until_quiet()

    def _set_technology(self, technology):
        """Switches the active simulation technology implementation."""
        if technology == CellularTechnology.LTE:
            self.iperf = Cmw500IperfMeasurement(self.simulator.cmw)
            self.tx_measurement = self.simulator.cmw.init_lte_measurement()

            # store lte_simulation since configuration is cleared when initializing
            if not self._lte_simulation:
                self._lte_simulation = CrOSLteSimulation.CrOSLteSimulation(
                    self.simulator,
                    self.logger,
                    self.dut,
                    {"attach_retries": 3, "attach_timeout": 60},
                    None,
                    power_mode=CrOSLteSimulation.PowerMode.RSRP,
                )
            self.simulation = self._lte_simulation
        elif technology == CellularTechnology.WCDMA:
            # no measurement/simulation available for WCDMA
            self.iperf = None
            self.tx_measurement = None
            self.simulation = None
        else:
            raise ValueError(f"Unsupported technology for CMW500: {technology}")

        self.technology = technology


class CMX500Configuration(CallboxConfiguration):
    """Configuration for CMX500 callboxes."""

    def __init__(self, dut, host, port, technology):
        """Initializes CMX500 controller configuration.

        Args:
            dut: a device handler implementing BaseCellularDut.
            host: the ip/host address of the callbox.
            port: the port number for the callbox controller commands.
            technology: the RAT technology.
        """
        super().__init__()
        self._pool = ThreadPool(processes=1)
        self._unsub_connect_event = None
        self._lte_simulation = None
        self._nr5g_nsa_simulation = None
        self.dut = dut
        self.host = host
        self.port = port
        self.technology = technology
        self.logger = logger.create_logger()
        self.simulator = cmx.CMX500CellularSimulator(
            host, port, config_mode=None
        )
        self.handover = Cmx500HandoverSimulator(self.simulator.cmx)
        self._set_technology(technology)
        cmx_tx.delete_all_multievals()

    def nr5g_sa_handover(self, band, channel, bw):
        """Performs a handover to 5G SA.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
            bw: the downlink bandwidth of the handover destination
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.nr5g_sa_handover(
            band, channel, bw, _HANDOVER_MAP[self.technology]
        )
        self._set_technology(CellularTechnology.NR5G_SA)

    def nr5g_nsa_handover(
        self,
        band,
        channel,
        bandwidth,
        secondary_band,
        secondary_channel,
        secondary_bandwidth,
    ):
        """Performs a handover to 5G NSA.

        Args:
            band: the band of the handover destination primary cell.
            channel: the downlink channel of the handover destination primary
                cell.
            bandwidth: the downlink bandwidth of the handover destination
                primary cell.
            secondary_band: the band of the handover destination secondary cell.
            secondary_channel: the downlink channel of the handover destination
                secondary cell.
            secondary_bandwidth: the downlink bandwidth of the handover
                destination secondary cell.
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.nr5g_nsa_handover(
            band,
            channel,
            bandwidth,
            secondary_band,
            secondary_channel,
            secondary_bandwidth,
            _HANDOVER_MAP[self.technology],
        )
        self._set_technology(CellularTechnology.NR5G_NSA)

    def lte_handover(self, band, channel, bw):
        """Performs a handover to LTE.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
            bw: the downlink bandwidth of the handover destination
        """
        if not self.technology in _HANDOVER_MAP:
            raise ValueError(
                f"Handover from {self.technology} is not supported in the current callbox configuration"
            )

        self.handover.lte_handover(
            band, channel, bw, _HANDOVER_MAP[self.technology]
        )
        self._set_technology(CellularTechnology.LTE)

    def wcdma_handover(self, band, channel):
        """Performs a handover to WCDMA.

        Args:
            band: the band of the handover destination
            channel: the downlink channel of the handover destination
        """
        # WCDMA handover not supported on CMX
        raise NotImplementedError()

    def configure(self, parameters):
        """Configures the callbox with the provided parameters.

        Args:
            parameters: the callbox configuration dictionary.
        """
        # no need to stop cmx before configuring like cmw, it's already
        # handled by cmx controller
        self._stop_connection_watcher()
        self.simulation.configure(parameters)
        # if SA, keep rrc otherwise will fall back to RRC_IDLE
        if self.technology == CellularTechnology.NR5G_SA:
            self.simulator.cmx.set_keep_rrc(True)
        self.simulator.wait_until_quiet()

        self._init_tx_meas()

    def start(self):
        """Starts a simulation on the callbox."""
        self._stop_connection_watcher()
        self.simulation.start()
        self.simulator.wait_until_quiet()

        # CMX500 won't automatically reattach secondary cells (ENDC or SCCs),
        # so watch for register/connection event and automatically attach
        # secondary carriers when device reconnects.
        if self.technology == CellularTechnology.NR5G_SA:
            self._unsub_connect_event = cmx500_events.on_fiveg_pdu_session_activate(
                # Launch reattach in background thread otherwise we will block
                # the XLAPI event thread and will timeout instead.
                lambda: self._pool.apply_async(
                    self._reattach_secondary_carriers
                )
            )
        else:
            self._unsub_connect_event = cmx500_events.on_emm_registered(
                lambda: self._pool.apply_async(
                    self._reattach_secondary_carriers
                )
            )

        # Reinitialize tx_measure since the primary cell may have changed.
        self._init_tx_meas()

    def close(self):
        """Releases any resources held open by the controller."""
        self.simulator.destroy()
        self._stop_connection_watcher()
        self._pool.close()

    def _set_technology(self, technology):
        """Switches the active simulation technology implementation."""
        self.iperf = None
        if technology == CellularTechnology.LTE:
            if not self._lte_simulation:
                self._lte_simulation = CrOSLteSimulation.CrOSLteSimulation(
                    self.simulator,
                    self.logger,
                    self.dut,
                    {"attach_retries": 1, "attach_timeout": 120},
                    None,
                    power_mode=CrOSLteSimulation.PowerMode.Total,
                )
            self.simulation = self._lte_simulation
        elif technology in (
            CellularTechnology.NR5G_NSA,
            CellularTechnology.NR5G_SA,
        ):
            if not self._nr5g_nsa_simulation:
                self._nr5g_nsa_simulation = CrOSLteSimulation.CrOSLteSimulation(
                    self.simulator,
                    self.logger,
                    self.dut,
                    {"attach_retries": 1, "attach_timeout": 120},
                    None,
                    nr_mode="nr",
                    power_mode=CrOSLteSimulation.PowerMode.Total,
                )
            self.simulation = self._nr5g_nsa_simulation
        else:
            raise ValueError(f"Unsupported technology for CMW500: {technology}")

        self.technology = technology

        # Reinitialize tx_measure since the primary cell may change.
        self._init_tx_meas()

    def _init_tx_meas(self):
        cell = self.simulator.cmx.primary_cell
        if cell:
            self.tx_measurement = cmx_tx.create_measurement(cell.cell)
        else:
            self.tx_measurement = None

    def _stop_connection_watcher(self):
        if self._unsub_connect_event:
            self._unsub_connect_event()
            self._unsub_connect_event = None

    def _reattach_secondary_carriers(self):
        """Reattach secondary carriers (includes ENDC and SCCs)."""
        if not self.simulator.cmx.secondary_cells:
            return

        self.logger.info(
            "Device reconnection detected, reattaching secondary carriers"
        )

        # lte_attach_secondary_carriers is used for LTE, NSA, and SA.
        self.simulator.lte_attach_secondary_carriers(None)
