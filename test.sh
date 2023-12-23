#!/bin/env
#assume this script will be executed in a git project
 URL=$(cat .git/config | grep url| cut -d\  -f3)
 rm -rf .git
 git init
 git add .
 git commit -m "Initial commit"
 git remote add origin $URL
 git push -u --force origin main
