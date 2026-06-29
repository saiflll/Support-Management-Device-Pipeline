import csv
import os
from datetime import datetime
import time

try:
    import psycopg2
except ImportError:
    print("Error: Library 'psycopg2' belum terinstall.")
    print("Silakan install dengan menjalankan: python -m pip install psycopg2-binary")
    exit(1)

# Konfigurasi Database
# Disamakan dengan environment default di main.go, tetapi Host default ke localhost untuk running di host
DB_HOST = os.getenv("DB_HOST", "172.20.100.11")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASS = os.getenv("DB_PASSWORD", "pPa3PLan")
DB_NAME = os.getenv("DB_NAME", "servfi")

# Konfigurasi Export
TARGET_PREFIX = "mdcw4"
START_DATE = "2026-01-01 00:00:00"
END_DATE = "2026-01-31 23:59:59"
OUTPUT_FILE = f"export_{TARGET_PREFIX}_jan2026.csv"

def export_data():
    print(f"[*] Menghubungkan ke database {DB_NAME} di {DB_HOST}:{DB_PORT}...")
    
    conn = None
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASS,
            database=DB_NAME
        )
        print("[+] Berhasil terhubung ke database.")

        cursor = conn.cursor()

        # Query untuk mengambil data
        query = """
            SELECT id, ts, reg2, reg5, reg114, prefix, created_at
            FROM production_mdcw
            WHERE prefix = %s
            AND created_at BETWEEN %s AND %s
            ORDER BY created_at ASC
        """
        
        print(f"[*] Menjalankan query untuk prefix '{TARGET_PREFIX}' dari {START_DATE} sampai {END_DATE}...")
        cursor.execute(query, (TARGET_PREFIX, START_DATE, END_DATE))
        
        rows = cursor.fetchall()
        print(f"[+] Ditemukan {len(rows)} baris data.")

        if len(rows) > 0:
            print(f"[*] Menulis ke file {OUTPUT_FILE}...")
            with open(OUTPUT_FILE, mode='w', newline='', encoding='utf-8') as csvfile:
                fieldnames = ['ID', 'TS (Device)', 'Reg2 (Pack Count)', 'Reg5 (Status)', 'Reg114 (Weight)', 'Prefix', 'Created At']
                writer = csv.writer(csvfile)
                writer.writerow(fieldnames)

                for row in rows:
                    # row: (id, ts, reg2, reg5, reg114, prefix, created_at)
                    # Format weight (reg114) bagi 10 seperti di main.go
                    weight_raw = row[4]
                    weight_formatted = f"{weight_raw / 10:.1f}".replace('.', ',')
                    
                    data_row = [
                        row[0], # ID
                        row[1], # TS
                        row[2], # Reg2
                        row[3], # Reg5
                        weight_formatted, # Reg114 (Formatted)
                        row[5], # Prefix
                        row[6]  # Created At
                    ]
                    writer.writerow(data_row)
            
            print(f"[+] Export berhasil! File tersimpan di: {os.path.abspath(OUTPUT_FILE)}")
        else:
            print("[-] Tidak ada data yang ditemukan untuk range tanggal tersebut.")

    except psycopg2.Error as e:
        print(f"[-] Database Error: {e}")
    except Exception as e:
        print(f"[-] Error: {e}")
    finally:
        if conn:
            conn.close()
            print("[*] Koneksi database ditutup.")

if __name__ == "__main__":
    export_data()
