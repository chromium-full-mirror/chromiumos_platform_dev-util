# this module will be imported in the into your flowgraph
from enum import Enum

class Noise(Enum):
	NOTHING=0	# Carrier OFF
	AWGN=1		# AWGN
	COS1=2		# +/-1MHz Modulated Sine waves
	COS5=3		# +/-5MHz Modulated Sine waves
	
	def __int__(self):
		return self.value

center_freq_list_MHz = [2442.0,5150.0,5350.0,5945.0,6525.0,7125.0]
samp_rate_matrix_MHz = [1,15,2,10]
IF_gain_matrix = \
		[[0,	0,	0,	0,	0,	0],	# Carrier OFF
		[47,	47,	47,	47,	47,	47],	# AWGN
		[45,	43,	44,	41,	42,	43],	# +/-1MHz Modulated Sine waves
		[32,	31,	34,	38,	42,	43]]	# +/-5MHz Modulated Sine waves
		#2442.0	5150.0	5350.0	5945.0	6525.0	7125.0	# Frequency
RF_gain_matrix = \
		[[0,	0,	0,	0,	0,	0],	# Carrier OFF
		[14,	14,	14,	14,	14,	14],	# AWGN
		[0,	14,	14,	14,	14,	14],	# +/-1MHz Modulated Sine waves
		[14,	14,	14,	14,	14,	14]]	# +/-5MHz Modulated Sine waves
		#2442.0	5150.0	5350.0	5945.0	6525.0	7125.0	# Frequency
RF_Pout_matrix_dBm_x10 = \
		[[-1740,-1740,	-1740,	-1740,	-1740,	-1740],	# Carrier OFF
		[126,	-129,	-127,	-142,	-267,	-289],	# AWGN
		[24,	-135,	-125,	-165,	-282,	-296],	# +/-1MHz Modulated Sine waves
		[12,	-266,	-237,	-216,	-304,	-317]]	# +/-5MHz Modulated Sine waves
		#2442.0	5150.0	5350.0	5945.0	6525.0	7125.0	# Frequency
		
def get_IF_gain(noise_type, frequency_index):
	return IF_gain_matrix[int(noise_type)][frequency_index]

def get_RF_gain(noise_type, frequency_index):
	return RF_gain_matrix[int(noise_type)][frequency_index]
	
def get_RF_Pout_x10(noise_type, frequency_index):
	return RF_Pout_matrix_dBm_x10[int(noise_type)][frequency_index]
	
def get_samp_rate(noise_type, frequency_index):
	return samp_rate_matrix_MHz[int(noise_type)]*1e6

def get_center_freq(center_freq_index):
	return center_freq_list_MHz[center_freq_index]*1e6
