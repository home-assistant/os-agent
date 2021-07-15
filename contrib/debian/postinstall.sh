#!/bin/bash
systemctl daemon-reload
systemctl enable haos-agent
systemctl start haos-agent
