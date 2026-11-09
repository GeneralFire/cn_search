export interface Car {
  id: number;
  photo?: string;
  address: string;
  plate: string; // госномер
  key: string;   // ключ (идентификатор)
  date: string;  // ISO-дата
}

export const cars: Car[] = [
  { id: 1, address: "Москва, Пресненская наб., 10", plate: "А123ВС 777", key: "KEY-0451", date: "2026-11-02" },
  { id: 2, address: "Санкт-Петербург, Невский пр., 28", plate: "М456ОР 178", key: "KEY-1187", date: "2026-10-27" },
  { id: 3, address: "Казань, ул. Баумана, 44", plate: "Т789УХ 116", key: "KEY-2093", date: "2026-11-08" },
];
