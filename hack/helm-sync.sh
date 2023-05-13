#! /bin/bash

echo "start sync helm chart to runtime-copilot-helm-chart repo".

git clone https://lengrongfu:${GITHUB_TOKEN}@github.com/lengrongfu/runtime-copilot-helm-charts.git

cd runtime-copilot-helm-charts

cp -rf ../charts ./

if git diff --quiet HEAD
then
    echo "Git项目没有发生变化"
    exit 0
fi

git add .
git commit -m "add helm chart"
git push https://lengrongfu:${GITHUB_TOKEN}@github.com/lengrongfu/runtime-copilot-helm-charts.git main