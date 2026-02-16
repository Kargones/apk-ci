#!/bin/bash
set -e

PKG=libwebkit2gtk-4.0-37
ARCH=amd64
SNAPSHOT_DATE=20240701  # дата, когда пакет ещё был в bookworm
MIRROR="http://snapshot.debian.org/archive/debian/${SNAPSHOT_DATE}/pool/main/w/webkit2gtk"

# Создадим временную папку
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

echo "[*] Скачиваем пакет $PKG для $ARCH с snapshot.debian.org ..."

# Получаем список доступных пакетов
wget -qO- "${MIRROR}/" | grep -oP "href=\"${PKG}_[^\"]+_${ARCH}\.deb\"" | sed 's/href="//;s/"//' > pkgs.txt

if [ ! -s pkgs.txt ]; then
  echo "[!] Не удалось найти пакет $PKG для $ARCH"
  exit 1
fi

# Берём последний по версии
PKG_FILE=$(tail -n1 pkgs.txt)

echo "[*] Скачиваем ${PKG_FILE}"
wget -q "${MIRROR}/${PKG_FILE}"

echo "[*] Устанавливаем пакет..."
dpkg -i "${PKG_FILE}" || apt -f install -y

echo "[+] Готово! Установлен пакет $PKG"
cd /
rm -rf "$TMPDIR"
