import psycopg2
from psycopg2 import sql
import os

# Konfigurasi Database (Sesuaikan dengan .env Anda jika berbeda)
# Jika dijalankan dari Docker, gunakan host container, misal: postgres_db
# Jika dari OS Host (Windows), gunakan localhost jika port diexpose, atau IP Docker Desktop
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_NAME = os.getenv("DB_NAME", "servfi")

# Data Akun berdasarkan init-rbac.sql
USERS = {
    "sani": "Sani123",
    "elec": "Elec123",
    "poss": "pos 123",
    "admin": "pPa3PLan" # atau postgres / password root Anda
}

def check_connection(user, password):
    """Fungsi untuk mencoba koneksi dan mengecek akses skema default."""
    print(f"\n=====================================")
    print(f"[*] Mengetes login dengan akun: {user}")
    
    conn = None
    try:
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            database=DB_NAME,
            user=user,
            password=password
        )
        conn.autocommit = True
        cursor = conn.cursor()
        print(f"[+] Berhasil login sebagai '{user}'")

        # 1. Cek User Saat Ini & Search Path
        cursor.execute("SELECT current_user, current_schemas(true);")
        user_info = cursor.fetchone()
        print(f"    - Current User: {user_info[0]}")
        print(f"    - Search Path: {user_info[1]}")

        # 2. Cek Akses Membuat Tabel di Skema Milik Sendiri
        # (Misal user 'sani' hanya boleh buat tabel di schema 'sani')
        try:
            if user != "admin": # Admin bebas
                table_name = f"test_table_{user}"
                cursor.execute(f"CREATE TABLE IF NOT EXISTS {user}.{table_name} (id SERIAL PRIMARY KEY, data TEXT);")
                cursor.execute(f"INSERT INTO {user}.{table_name} (data) VALUES ('Halo dari {user}');")
                cursor.execute(f"SELECT * FROM {user}.{table_name};")
                row = cursor.fetchone()
                print(f"[+] Sukses membuat tabel dan membaca data di schema '{user}': {row}")
                
                # Bersihkan (Opsional)
                cursor.execute(f"DROP TABLE {user}.{table_name};")
                print(f"[+] Sukses menghapus tabel uji coba di schema '{user}'.")
        except Exception as e:
            print(f"[-] Gagal beroperasi di schema sendiri: {e}")
            conn.rollback()

        # 3. Cek Akses ke Tabel Public (production_mdcw)
        try:
            cursor.execute("SELECT COUNT(*) FROM public.production_mdcw;")
            count = cursor.fetchone()[0]
            print(f"[+] Akses ke 'public.production_mdcw' DIIZINKAN. Jumlah baris: {count}")
        except psycopg2.errors.InsufficientPrivilege:
            print(f"[-] Akses DITOLAK: Akun '{user}' tidak boleh membaca 'public.production_mdcw'. (Sudah sesuai RBAC)")
            conn.rollback()
        except Exception as e:
            print(f"[-] Error Public Table (Tabel mungkin belum ada): {e}")
            conn.rollback()

    except psycopg2.OperationalError as e:
        print(f"[-] Gagal login sebagai {user}. Alasan: {e}".strip())
    finally:
        if conn:
            cursor.close()
            conn.close()

if __name__ == "__main__":
    print(f"Memulai Pengecekan RBAC Database di {DB_HOST}:{DB_PORT} / {DB_NAME}")
    
    # 1. Test akun sani (Seharusnya tidak bisa baca public.production_mdcw)
    check_connection("sani", USERS["sani"])
    
    # 2. Test akun elec (Seharusnya tidak bisa baca public.production_mdcw)
    check_connection("elec", USERS["elec"])
    
    # 3. Test akun poss (Seharusnya BISA baca public.production_mdcw)
    check_connection("poss", USERS["poss"])
