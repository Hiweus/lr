# lr

Helper to execute lambda

## Installing
```shell
CURRENT_UID=$(id -u):$(id -g) docker compose up --build


# Choose one
echo PATH=$PATH:$HOME/.local/bin >> $HOME/.bashrc
echo PATH=$PATH:$HOME/.local/bin >> $HOME/.zshrc

mkdir -p $HOME/.local/bin
mv lr $HOME/.local/bin/lr

# Choose one
. $HOME/.bashrc
. $HOME/.zshrc
```