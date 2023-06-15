#!/usr/bin/env python3
# -*- coding: utf-8 -*-

#
# SPDX-License-Identifier: GPL-3.0
#
# GNU Radio Python Flow Graph
# Title: New Calibration Controller
# GNU Radio version: 3.10.1.1

from packaging.version import Version as StrictVersion

if __name__ == '__main__':
    import ctypes
    import sys
    if sys.platform.startswith('linux'):
        try:
            x11 = ctypes.cdll.LoadLibrary('libX11.so')
            x11.XInitThreads()
        except:
            print("Warning: failed to XInitThreads()")

from PyQt5 import Qt
from PyQt5.QtCore import QObject, pyqtSlot
from gnuradio import eng_notation
from gnuradio import analog
from gnuradio import blocks
from gnuradio import gr
from gnuradio.filter import firdes
from gnuradio.fft import window
import sys
import signal
from argparse import ArgumentParser
from gnuradio.eng_arg import eng_float, intx
from gnuradio import network
from gnuradio import soapy
from xmlrpc.client import ServerProxy
import CalibrationController_vals as vals  # embedded python module
import os



from gnuradio import qtgui

class CalibrationController(gr.top_block, Qt.QWidget):

    def __init__(self):
        gr.top_block.__init__(self, "New Calibration Controller", catch_exceptions=True)
        Qt.QWidget.__init__(self)
        self.setWindowTitle("New Calibration Controller")
        qtgui.util.check_set_qss()
        try:
            self.setWindowIcon(Qt.QIcon.fromTheme('gnuradio-grc'))
        except:
            pass
        self.top_scroll_layout = Qt.QVBoxLayout()
        self.setLayout(self.top_scroll_layout)
        self.top_scroll = Qt.QScrollArea()
        self.top_scroll.setFrameStyle(Qt.QFrame.NoFrame)
        self.top_scroll_layout.addWidget(self.top_scroll)
        self.top_scroll.setWidgetResizable(True)
        self.top_widget = Qt.QWidget()
        self.top_scroll.setWidget(self.top_widget)
        self.top_layout = Qt.QVBoxLayout(self.top_widget)
        self.top_grid_layout = Qt.QGridLayout()
        self.top_layout.addLayout(self.top_grid_layout)

        self.settings = Qt.QSettings("GNU Radio", "CalibrationController")

        try:
            if StrictVersion(Qt.qVersion()) < StrictVersion("5.0.0"):
                self.restoreGeometry(self.settings.value("geometry").toByteArray())
            else:
                self.restoreGeometry(self.settings.value("geometry"))
        except:
            pass

        ##################################################
        # Variables
        ##################################################
        self.noise_source = noise_source = 0
        self.frequency = frequency = 0
        self.measured_power = measured_power = vals.get_RF_Pout_x10(noise_source,frequency)
        self.samp_rate = samp_rate = vals.get_samp_rate(noise_source,frequency)
        self.rf_amplifier = rf_amplifier = vals.get_RF_gain(noise_source,frequency)
        self.if_gain = if_gain = vals.get_IF_gain(noise_source,frequency)
        self.display = display = "{0} dBm".format(measured_power/10)
        self.center_freq = center_freq = vals.get_center_freq(frequency)

        ##################################################
        # Blocks
        ##################################################
        # Create the options list
        self._noise_source_options = [0, 1, 2, 3]
        # Create the labels list
        self._noise_source_labels = ['Nothing', 'AWGN', '1MHZ Cosine', '5MHZ Cosine']
        # Create the combo box
        # Create the radio buttons
        self._noise_source_group_box = Qt.QGroupBox("Signal Source" + ": ")
        self._noise_source_box = Qt.QVBoxLayout()
        class variable_chooser_button_group(Qt.QButtonGroup):
            def __init__(self, parent=None):
                Qt.QButtonGroup.__init__(self, parent)
            @pyqtSlot(int)
            def updateButtonChecked(self, button_id):
                self.button(button_id).setChecked(True)
        self._noise_source_button_group = variable_chooser_button_group()
        self._noise_source_group_box.setLayout(self._noise_source_box)
        for i, _label in enumerate(self._noise_source_labels):
            radio_button = Qt.QRadioButton(_label)
            self._noise_source_box.addWidget(radio_button)
            self._noise_source_button_group.addButton(radio_button, i)
        self._noise_source_callback = lambda i: Qt.QMetaObject.invokeMethod(self._noise_source_button_group, "updateButtonChecked", Qt.Q_ARG("int", self._noise_source_options.index(i)))
        self._noise_source_callback(self.noise_source)
        self._noise_source_button_group.buttonClicked[int].connect(
            lambda i: self.set_noise_source(self._noise_source_options[i]))
        self.top_grid_layout.addWidget(self._noise_source_group_box, 2, 0, 1, 1)
        for r in range(2, 3):
            self.top_grid_layout.setRowStretch(r, 1)
        for c in range(0, 1):
            self.top_grid_layout.setColumnStretch(c, 1)
        self.xmlrpc_client_0_0 = ServerProxy('http://'+'localhost'+':8080')
        self.xmlrpc_client_0 = ServerProxy('http://'+'localhost'+':8080')
        self.soapy_hackrf_sink_0 = None
        dev = 'driver=hackrf'
        stream_args = ''
        tune_args = ['']
        settings = ['']

        self.soapy_hackrf_sink_0 = soapy.sink(dev, "fc32", 1, '',
                                  stream_args, tune_args, settings)
        self.soapy_hackrf_sink_0.set_sample_rate(0, samp_rate)
        self.soapy_hackrf_sink_0.set_bandwidth(0, 0)
        self.soapy_hackrf_sink_0.set_frequency(0, center_freq)
        self.soapy_hackrf_sink_0.set_gain(0, 'AMP', True if rf_amplifier == 14 else 0)
        self.soapy_hackrf_sink_0.set_gain(0, 'VGA', min(max(if_gain, 0.0), 47.0))
        self.network_tcp_sink_0 = network.tcp_sink(gr.sizeof_gr_complex, 1, '127.0.0.1', 1234,1)
        # Create the options list
        self._frequency_options = [0, 1, 2, 3, 4, 5]
        # Create the labels list
        self._frequency_labels = ['2442', '5150', '5350', '5945', '6525', '7125']
        # Create the combo box
        # Create the radio buttons
        self._frequency_group_box = Qt.QGroupBox("Frequency (MHz)" + ": ")
        self._frequency_box = Qt.QHBoxLayout()
        class variable_chooser_button_group(Qt.QButtonGroup):
            def __init__(self, parent=None):
                Qt.QButtonGroup.__init__(self, parent)
            @pyqtSlot(int)
            def updateButtonChecked(self, button_id):
                self.button(button_id).setChecked(True)
        self._frequency_button_group = variable_chooser_button_group()
        self._frequency_group_box.setLayout(self._frequency_box)
        for i, _label in enumerate(self._frequency_labels):
            radio_button = Qt.QRadioButton(_label)
            self._frequency_box.addWidget(radio_button)
            self._frequency_button_group.addButton(radio_button, i)
        self._frequency_callback = lambda i: Qt.QMetaObject.invokeMethod(self._frequency_button_group, "updateButtonChecked", Qt.Q_ARG("int", self._frequency_options.index(i)))
        self._frequency_callback(self.frequency)
        self._frequency_button_group.buttonClicked[int].connect(
            lambda i: self.set_frequency(self._frequency_options[i]))
        self.top_grid_layout.addWidget(self._frequency_group_box, 2, 1, 1, 1)
        for r in range(2, 3):
            self.top_grid_layout.setRowStretch(r, 1)
        for c in range(1, 2):
            self.top_grid_layout.setColumnStretch(c, 1)
        self._display_tool_bar = Qt.QToolBar(self)

        if None:
            self._display_formatter = None
        else:
            self._display_formatter = lambda x: str(x)

        self._display_tool_bar.addWidget(Qt.QLabel("Output Power: "))
        self._display_label = Qt.QLabel(str(self._display_formatter(self.display)))
        self._display_tool_bar.addWidget(self._display_label)
        self.top_grid_layout.addWidget(self._display_tool_bar, 0, 0, 2, 2)
        for r in range(0, 2):
            self.top_grid_layout.setRowStretch(r, 1)
        for c in range(0, 2):
            self.top_grid_layout.setColumnStretch(c, 1)
        self.blocks_selector_0 = blocks.selector(gr.sizeof_gr_complex*1,noise_source,0)
        self.blocks_selector_0.set_enabled(True)
        self.analog_sig_source_x_1 = analog.sig_source_c(samp_rate, analog.GR_COS_WAVE, 5e6, 1, 0, 0)
        self.analog_sig_source_x_0 = analog.sig_source_c(samp_rate, analog.GR_COS_WAVE, 1e6, 1, 0, 0)
        self.analog_noise_source_x_0 = analog.noise_source_c(analog.GR_GAUSSIAN, 1, 0)
        self.analog_const_source_x_1 = analog.sig_source_c(0, analog.GR_CONST_WAVE, 0, 0, 0)


        ##################################################
        # Connections
        ##################################################
        self.connect((self.analog_const_source_x_1, 0), (self.blocks_selector_0, 0))
        self.connect((self.analog_noise_source_x_0, 0), (self.blocks_selector_0, 1))
        self.connect((self.analog_sig_source_x_0, 0), (self.blocks_selector_0, 2))
        self.connect((self.analog_sig_source_x_1, 0), (self.blocks_selector_0, 3))
        self.connect((self.blocks_selector_0, 0), (self.network_tcp_sink_0, 0))
        self.connect((self.blocks_selector_0, 0), (self.soapy_hackrf_sink_0, 0))


    def closeEvent(self, event):
        self.settings = Qt.QSettings("GNU Radio", "CalibrationController")
        self.settings.setValue("geometry", self.saveGeometry())
        self.stop()
        self.wait()

        event.accept()

    def setStyleSheetFromFile(self, filename):
        try:
            if not os.path.exists(filename):
                filename = os.path.join(
                    gr.prefix(), "share", "gnuradio", "themes", filename)
            with open(filename) as ss:
                self.setStyleSheet(ss.read())
        except Exception as e:
            print(e, file=sys.stderr)

    def get_noise_source(self):
        return self.noise_source

    def set_noise_source(self, noise_source):
        self.noise_source = noise_source
        self.set_if_gain(vals.get_IF_gain(self.noise_source,self.frequency))
        self.set_measured_power(vals.get_RF_Pout_x10(self.noise_source,self.frequency))
        self._noise_source_callback(self.noise_source)
        self.set_rf_amplifier(vals.get_RF_gain(self.noise_source,self.frequency))
        self.set_samp_rate(vals.get_samp_rate(self.noise_source,self.frequency))
        self.blocks_selector_0.set_input_index(self.noise_source)

    def get_frequency(self):
        return self.frequency

    def set_frequency(self, frequency):
        self.frequency = frequency
        self.set_center_freq(vals.get_center_freq(self.frequency))
        self._frequency_callback(self.frequency)
        self.set_if_gain(vals.get_IF_gain(self.noise_source,self.frequency))
        self.set_measured_power(vals.get_RF_Pout_x10(self.noise_source,self.frequency))
        self.set_rf_amplifier(vals.get_RF_gain(self.noise_source,self.frequency))
        self.set_samp_rate(vals.get_samp_rate(self.noise_source,self.frequency))

    def get_measured_power(self):
        return self.measured_power

    def set_measured_power(self, measured_power):
        self.measured_power = measured_power
        self.set_display("{0} dBm".format(self.measured_power/10))

    def get_samp_rate(self):
        return self.samp_rate

    def set_samp_rate(self, samp_rate):
        self.samp_rate = samp_rate
        self.analog_sig_source_x_0.set_sampling_freq(self.samp_rate)
        self.analog_sig_source_x_1.set_sampling_freq(self.samp_rate)
        self.soapy_hackrf_sink_0.set_sample_rate(0, self.samp_rate)
        self.xmlrpc_client_0.set_samp_rate(self.samp_rate)

    def get_rf_amplifier(self):
        return self.rf_amplifier

    def set_rf_amplifier(self, rf_amplifier):
        self.rf_amplifier = rf_amplifier
        self.soapy_hackrf_sink_0.set_gain(0, 'AMP', True if self.rf_amplifier == 14 else 0)

    def get_if_gain(self):
        return self.if_gain

    def set_if_gain(self, if_gain):
        self.if_gain = if_gain
        self.soapy_hackrf_sink_0.set_gain(0, 'VGA', min(max(self.if_gain, 0.0), 47.0))

    def get_display(self):
        return self.display

    def set_display(self, display):
        self.display = display
        Qt.QMetaObject.invokeMethod(self._display_label, "setText", Qt.Q_ARG("QString", str(self._display_formatter(self.display))))

    def get_center_freq(self):
        return self.center_freq

    def set_center_freq(self, center_freq):
        self.center_freq = center_freq
        self.soapy_hackrf_sink_0.set_frequency(0, self.center_freq)
        self.xmlrpc_client_0_0.set_center_freq(self.center_freq)




def main(top_block_cls=CalibrationController, options=None):

    if StrictVersion("4.5.0") <= StrictVersion(Qt.qVersion()) < StrictVersion("5.0.0"):
        style = gr.prefs().get_string('qtgui', 'style', 'raster')
        Qt.QApplication.setGraphicsSystem(style)
    qapp = Qt.QApplication(sys.argv)

    tb = top_block_cls()

    tb.start()

    tb.setStyleSheetFromFile("./plain2.qss")
    tb.show()

    def sig_handler(sig=None, frame=None):
        tb.stop()
        tb.wait()

        Qt.QApplication.quit()

    signal.signal(signal.SIGINT, sig_handler)
    signal.signal(signal.SIGTERM, sig_handler)

    timer = Qt.QTimer()
    timer.start(500)
    timer.timeout.connect(lambda: None)

    qapp.exec_()

if __name__ == '__main__':
    main()
