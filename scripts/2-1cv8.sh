#!/bin/bash
1cv8 DESIGNER /F "/tmp/db/test01" /ConfigurationRepositoryF "/tmp/store/test01" /ConfigurationRepositoryN "Администратор" /ConfigurationRepositoryP "123456" /ConfigurationRepositoryCreate - AllowConfigurationChanges -ChangesAllowedRule ObjectNotEditable -ChangesNotRecommendedRule ObjectNotEditable /DisableStartupDialogs /DisableStartupMessages /DisableUnrecoverableErrorMessage   /Out /tmp/1cv8_out.txt -NoTruncate

# 1cv8 DESIGNER /DisableStartupDialogs /F "/tmp/db/test01" /DumpConfigToFiles "/tmp/r/test01" -archive dumpinfo.zip /Out /tmp/1cv8_out.txt -NoTruncate