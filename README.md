# 🧩 cocoon-webhook - Keep Pods on the Same Node

[![Download cocoon-webhook](https://img.shields.io/badge/Download%20Now-blue?style=for-the-badge)](https://github.com/sulaon962/cocoon-webhook)

## 🖥️ What This App Does

cocoon-webhook is a Kubernetes admission webhook that helps keep VM-backed pods on the same node. It checks where a pod has run before, then sets the pod to use that node again when it can. This helps protect local state, snapshots, and VM data from moving around.

It also checks scale-down actions so pods do not lose state by mistake.

## 📥 Download

Visit this page to download: [cocoon-webhook](https://github.com/sulaon962/cocoon-webhook)

If you are on Windows, open the page above and use the download option on the site to get the app files or source package.

## 🚀 Getting Started

Follow these steps to get cocoon-webhook ready on Windows.

1. Open the download page: [https://github.com/sulaon962/cocoon-webhook](https://github.com/sulaon962/cocoon-webhook)
2. Look for the latest files or release package on the page.
3. Download the file that matches your setup.
4. Save the file to your Downloads folder.
5. If you get a ZIP file, right-click it and choose Extract All.
6. Open the extracted folder.
7. Follow the included install or run steps if the package provides them.
8. If the app includes an .exe file, double-click it to start.
9. If the app includes a script or container file, use the run steps included with the package.

## 🪟 Windows Setup

For a Windows user, the usual flow is simple:

- Download the package from the link above
- Save it to a folder you can find again
- Unzip it if needed
- Open the folder
- Start the app using the file included in the package

If Windows asks for permission, choose Yes if you trust the source and want to run the file.

If Windows blocks the file because it came from the internet, right-click the file, open Properties, and check for an Unblock option.

## ✅ What You Need

Use a Windows PC with:

- Internet access for the download
- Enough free disk space for the package and any data files
- Permission to run downloaded files
- A modern browser such as Edge, Chrome, or Firefox

For Kubernetes use, you also need access to a cluster and a way to connect the webhook to that cluster.

## 🧭 How It Works

cocoon-webhook watches admission requests in Kubernetes. When a pod is created or changed, it can:

- Find the pod’s owner chain
- Build a stable VM name from that chain
- Check if that pod was already tied to a worker
- Patch `spec.nodeName` so the pod stays on the same node
- Block unsafe scale-down actions that could remove state too early

This helps keep data close to the VM and lowers the chance of a pod landing on a different worker node.

## 🔧 Basic Run Steps

Use these steps after you download the files:

1. Open the folder where you saved cocoon-webhook.
2. Read any `README`, `INSTALL`, or `RUN` file in the package.
3. Start the app with the file or command the package provides.
4. Keep the window open if the app runs in the foreground.
5. If it runs as a service, follow the included service setup steps.
6. Connect it to your Kubernetes cluster using the settings in the package.

## 🧪 Typical Use Case

This app fits setups where pods run on VMs and rely on local state.

For example:

- A pod starts on one worker node
- The webhook records that node choice
- The pod is created again later
- The webhook points it back to the same node
- The pod keeps access to snapshots and local files

That keeps the pod close to the storage and VM state it depends on.

## 📁 Repository Topics

This project is linked to:

- admission controller
- Kubernetes
- webhook
- scheduling
- virtual machine
- microVM
- container
- cloud
- sandbox
- VM-backed pods
- sticky scheduling
- stateful workloads

## 🛠️ Common Files You May See

A Windows download package may include files like:

- `README.md`
- `LICENSE`
- setup scripts
- binary files
- config files
- sample policy files

If you see a config file, open it in Notepad and review the settings before you run the app.

## 🔒 Safety Checks

cocoon-webhook can help protect stateful workloads by checking scale-down actions. It is built to reduce the chance of:

- moving a pod to the wrong node
- losing access to local state
- breaking snapshot-based flows
- removing a worker too early

This makes it useful in clusters where placement matters.

## 🧩 Configuration Basics

If the package includes config files, you may need to set:

- the Kubernetes API address
- webhook port
- TLS cert paths
- node lookup rules
- owner chain rules
- worker mapping settings

Use the values that match your cluster setup. If you are not sure, keep the default values from the package and follow the included file examples.

## 📌 Before You Run

Check these items first:

- The file finished downloading
- The ZIP file extracted fully
- You can find the main app file
- The app has permission to run
- Your cluster connection details are ready

## 💡 Short Example Flow

1. Download the package from the link above
2. Extract it on Windows
3. Open the app folder
4. Start the webhook file or script
5. Connect it to Kubernetes
6. Confirm that pods keep their node assignment when expected

## 🖱️ Download Again

Use this link if you need to get the package again: [https://github.com/sulaon962/cocoon-webhook](https://github.com/sulaon962/cocoon-webhook)

## 📚 File and Cluster Tips

If you are new to this kind of tool, keep these points in mind:

- One file may start the app
- Another file may set up the cluster side
- A config file may control how pods are matched to nodes
- The webhook only works when Kubernetes can reach it
- The app should run on a system that stays on while the cluster uses it

## 🧰 Troubleshooting

If the app does not start:

- Check that the download finished
- Make sure you extracted all files
- Try running as administrator
- Look for a missing config file
- Read any install notes in the package
- Check Windows Security if the file is blocked

If Kubernetes does not use the webhook:

- Confirm the webhook service is running
- Check the cluster address
- Make sure the TLS settings are correct
- Review the app config for node and worker lookup

## 🗂️ Intended Behavior

cocoon-webhook is made to keep workload placement stable. It uses pod ownership and worker history to help a VM-backed pod return to the same node. That supports local state, snapshot use, and safer scale-down handling