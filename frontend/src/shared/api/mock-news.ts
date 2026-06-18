// ═══ World News Mock Data ═══
// Mock data and type definitions for the World Finance News Dashboard.

export interface StockMarketData {
  indexName: string;
  value: string;
  change: string;
  changePercent: string;
  isPositive: boolean;
  chartPoints: number[];
  chartLabels: string[];
  thumbnail: string;
  highlights: string[];
  cnbcUrl: string;
  wsjUrl: string;
  reutersUrl: string;
}

export interface OilMarketData {
  wtiPrice: string;
  wtiChange: string;
  wtiPercent: string;
  wtiPositive: boolean;
  brentPrice: string;
  brentChange: string;
  brentPercent: string;
  brentPositive: boolean;
  chartPointsWTI: number[];
  chartPointsBrent: number[];
  chartLabels: string[];
  thumbnail: string;
  highlights: string[];
  reutersUrl: string;
  nytimesUrl: string;
  wsjUrl: string;
}

export interface GoldUSDData {
  goldPrice: string;
  goldChange: string;
  goldPercent: string;
  goldPositive: boolean;
  dxyPrice: string;
  dxyChange: string;
  dxyPercent: string;
  dxyPositive: boolean;
  chartPointsGold: number[];
  chartPointsDXY: number[];
  chartLabels: string[];
  highlights: string[];
  reutersUrl: string;
}

export interface NewsArticle {
  title: string;
  summary: string;
  source: string;
  time: string;
  url?: string;
  logo?: string;
}

export interface BreakingNews {
  time: string;
  source: string;
  content: string;
  isUrgent: boolean;
}

export interface WatchlistItem {
  event: string;
  time: string;
  importance: 'high' | 'medium' | 'low';
  source: string;
}

export interface WorldNewsReport {
  date: string; // YYYY-MM-DD
  quickHighlights: string[]; // Tóm tắt đầu bài
  keyNumbers: {
    label: string;
    value: string;
    change: string;
    isPositive: boolean;
    sparkline: number[];
  }[];
  stocks: StockMarketData;
  oil: OilMarketData;
  goldUsd: GoldUSDData;
  vtvIndexNews: NewsArticle[];
  vietnamFinanceNews: NewsArticle[];
  breakingNews: BreakingNews[];
  watchlist: WatchlistItem[];
}

export const MOCK_NEWS_DATA: Record<string, WorldNewsReport> = {
  '2026-06-17': {
    date: '2026-06-17',
    quickHighlights: [
      'Chứng khoán Mỹ chốt phiên tăng mạnh nhờ sự bùng nổ của nhóm cổ phiếu công nghệ lớn, dẫn dắt bởi NVIDIA vượt Microsoft trở thành công ty giá trị nhất thế giới.',
      'Giá dầu thô WTI và Brent tiếp tục xu hướng đi lên sau báo cáo tồn kho dầu của Mỹ giảm mạnh hơn dự báo và lo ngại căng thẳng leo thang ở Trung Đông.',
      'Vàng thế giới ổn định quanh mốc $2.330/ounce khi các nhà đầu tư chờ đợi số liệu kinh tế quan trọng để đoán định bước đi tiếp theo của Fed.',
      'Tỷ giá USD hạ nhiệt nhẹ sau số liệu bán lẻ của Mỹ yếu hơn dự đoán, dấy lên hy vọng Fed sẽ cắt giảm lãi suất sớm hơn.'
    ],
    keyNumbers: [
      { label: 'S&P 500', value: '5,487.02', change: '+1.02%', isPositive: true, sparkline: [5432, 5440, 5435, 5450, 5462, 5455, 5475, 5487] },
      { label: 'Dầu Brent', value: '$85.33', change: '+1.28%', isPositive: true, sparkline: [84.1, 84.3, 84.0, 84.6, 84.9, 84.7, 85.1, 85.33] },
      { label: 'Vàng thế giới', value: '$2,332.10', change: '+0.41%', isPositive: true, sparkline: [2321, 2325, 2320, 2328, 2331, 2326, 2330, 2332.1] }
    ],
    stocks: {
      indexName: 'S&P 500 & Nasdaq Composite',
      value: '5,487.02 / 17,862.23',
      change: '+55.22 / +214.30',
      changePercent: '+1.02% / +1.21%',
      isPositive: true,
      chartPoints: [5430, 5445, 5438, 5452, 5468, 5459, 5478, 5487],
      chartLabels: ['9:30', '10:30', '11:30', '12:30', '13:30', '14:30', '15:30', '16:00'],
      thumbnail: '/images/cnbc_trader.png',
      highlights: [
        'Nhóm công nghệ dẫn sóng: Cổ phiếu NVIDIA (NVDA) tăng thêm 3.5%, nâng vốn hóa lên mức kỷ lục 3.34 nghìn tỷ USD, chính thức vượt qua Microsoft để đứng vị trí số 1 toàn cầu.',
        'Nhận định từ Wall Street Journal: Đà tăng được củng cố khi lợi suất trái phiếu chính phủ Mỹ kỳ hạn 10 năm giảm xuống mức 4.22% nhờ dữ liệu lạm phát hạ nhiệt gần đây.',
        'Diễn biến châu Âu và châu Á: Chỉ số Stoxx 600 châu Âu tăng 0.7%, trong khi Nikkei 225 Nhật Bản chốt phiên tăng 1.1% nhờ kỳ vọng xuất khẩu phục hồi.'
      ],
      cnbcUrl: 'https://www.cnbc.com/world/',
      wsjUrl: 'https://www.wsj.com/finance/stocks?mod=nav_top_subsection',
      reutersUrl: 'https://www.reuters.com/markets/stocks/'
    },
    oil: {
      wtiPrice: '$81.57',
      wtiChange: '+$1.13',
      wtiPercent: '+1.40%',
      wtiPositive: true,
      brentPrice: '$85.33',
      brentChange: '+$1.08',
      brentPercent: '+1.28%',
      brentPositive: true,
      chartPointsWTI: [80.3, 80.6, 80.4, 80.9, 81.1, 80.9, 81.4, 81.57],
      chartPointsBrent: [84.1, 84.4, 84.2, 84.7, 84.9, 84.6, 85.1, 85.33],
      chartLabels: ['9:00', '11:00', '13:00', '15:00', '17:00', '19:00', '21:00', '23:00'],
      thumbnail: '/images/reuters_oil.png',
      highlights: [
        'Dữ liệu tồn kho: Báo cáo sơ bộ của Viện Dầu khí Mỹ (API) cho thấy dự trữ dầu thô thương mại giảm 4.4 triệu thùng trong tuần trước, mức giảm mạnh hơn nhiều so với dự báo của giới phân tích.',
        'Sàn giao dịch: Phiên chốt WTI tại NYMEX lúc 16h đạt đỉnh mới trong tháng, trong khi dầu Brent sàn ICE chốt phiên sát nút $85.5 nhờ nhu cầu tiêu thụ mùa hè dồi dào.',
        'Rủi ro Trung Đông: Xung đột biên giới giữa Israel và lực lượng Hezbollah ở Lebanon leo thang làm dấy lên mối lo ngại về khả năng gián đoạn nguồn cung tại khu vực sản xuất dầu mỏ lớn nhất thế giới.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/',
      nytimesUrl: 'https://www.nytimes.com/section/business/energy-environment?page=2',
      wsjUrl: 'https://www.wsj.com/business/energy-oil?mod=nav_top_subsection'
    },
    goldUsd: {
      goldPrice: '$2,332.10',
      goldChange: '+$9.50',
      goldPercent: '+0.41%',
      goldPositive: true,
      dxyPrice: '105.25',
      dxyChange: '-0.27',
      dxyPercent: '-0.26%',
      dxyPositive: false,
      chartPointsGold: [2322, 2326, 2320, 2328, 2332, 2327, 2329, 2332.1],
      chartPointsDXY: [105.52, 105.45, 105.48, 105.35, 105.28, 105.32, 105.24, 105.25],
      chartLabels: ['8:00', '10:00', '12:00', '14:00', '16:00', '18:00', '20:00', '22:00'],
      highlights: [
        'Giá Vàng thế giới tăng nhẹ trở lại sau số liệu Doanh thu Bán lẻ của Mỹ tháng 5 chỉ tăng 0.1% (thấp hơn kỳ vọng 0.3%). Điều này hỗ trợ các tài sản không hưởng lãi suất như vàng do củng cố giả thuyết Fed sắp giảm chi phí vay.',
        'Chỉ số DXY tiếp tục suy yếu nhẹ từ vùng đỉnh 105.8. Tuy nhiên, đà giảm được hạn chế khi các đồng tiền lớn đối thủ như EUR và GBP cũng đang chịu áp lực mất giá do bất ổn bầu cử tại châu Âu.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/'
    },
    vtvIndexNews: [
      {
        title: 'Châu Âu thận trọng trước thềm bầu cử sớm tại Pháp',
        summary: 'Các nhà phân tích kinh tế tại Brussels cảnh báo những biến động chính trị tại Pháp có thể gây áp lực lên trái phiếu khu vực Eurozone.',
        source: 'VTVIndex Thế giới',
        time: '14 giờ trước'
      },
      {
        title: 'Thương mại Mỹ - Trung: Căng thẳng mới về pin xe điện',
        summary: 'Bộ Thương mại Mỹ vừa công bố các biện pháp siết chặt kiểm tra nguồn gốc khoáng sản quan trọng nhập khẩu dùng cho pin xe điện.',
        source: 'VTVIndex Thế giới',
        time: '19 giờ trước'
      },
      {
        title: 'Chính sách tiền tệ Nhật Bản: BOJ chưa vội giảm mua trái phiếu',
        summary: 'Ngân hàng Trung ương Nhật Bản phát tín hiệu sẽ kéo dài lộ trình thắt chặt định lượng, làm suy yếu đồng Yên nội tệ so với USD.',
        source: 'VTVIndex Thế giới',
        time: '23 giờ trước'
      }
    ],
    vietnamFinanceNews: [
      {
        title: 'Kinh tế vĩ mô quý II: Động lực phục hồi vững chắc từ xuất khẩu',
        summary: 'Ấn phẩm mới của Tạp chí Kinh tế Sài Gòn nhấn mạnh tốc độ tăng trưởng xuất khẩu linh kiện điện tử và dệt may đã bù đắp phần nào tiêu dùng nội địa yếu.',
        source: 'Thời báo Kinh tế Sài Gòn',
        time: '2 giờ trước'
      },
      {
        title: 'Doanh nghiệp công nghệ chật vật giữ chân nhân sự AI',
        summary: 'Theo khảo sát của VNeconomy, làn sóng đầu tư vào AI tại các ngân hàng lớn đang đẩy mức lương kỹ sư dữ liệu lên cao kỷ lục, gây khó khăn cho startup.',
        source: 'VNeconomy',
        time: '5 giờ trước'
      },
      {
        title: 'Thị trường vàng miếng SJC hạ nhiệt nhanh chóng nhờ phương án bán trực tiếp',
        summary: 'Vietstock ghi nhận chênh lệch giữa giá vàng miếng trong nước và thế giới thu hẹp về dưới 5 triệu đồng/lượng nhờ nỗ lực điều tiết của Ngân hàng Nhà nước.',
        source: 'Vietstock',
        time: '8 giờ trước'
      },
      {
        title: 'Tỷ giá VND/USD chịu áp lực lớn trước phiên họp thường niên của Fed',
        summary: 'Diễn đàn Doanh nghiệp phân tích dòng vốn đầu tư gián tiếp nước ngoài đảo chiều và nhu cầu ngoại tệ thanh toán xăng dầu khiến tỷ giá liên ngân hàng duy trì sát biên độ trần.',
        source: 'Báo Diễn đàn Doanh nghiệp',
        time: '12 giờ trước'
      }
    ],
    breakingNews: [
      { time: '06:45', source: 'Bloomberg', content: 'Bộ Tư pháp Mỹ (DOJ) chuẩn bị mở cuộc điều tra chống độc quyền mới đối với thỏa thuận mua lại chip AI của NVIDIA.', isUrgent: true },
      { time: '06:15', source: 'CNBC', content: 'Chỉ số tương lai S&P 500 tăng nhẹ 0.15% trước giờ mở cửa phiên giao dịch châu Á hôm nay.', isUrgent: false },
      { time: '05:30', source: 'Reuters', content: 'Tổng thống Nga và Nhà lãnh đạo Triều Tiên ký hiệp ước đối tác chiến lược toàn diện, gia tăng căng thẳng địa chính trị.', isUrgent: true }
    ],
    watchlist: [
      { event: 'Công bố Chỉ số CPI Anh Quốc (Tháng 5)', time: '13:00 hôm nay', importance: 'high', source: 'Financial Times' },
      { event: 'Doanh số bán nhà xây mới tại Mỹ (Tháng 5)', time: '21:00 hôm nay', importance: 'medium', source: 'Bloomberg' },
      { event: 'Họp chính sách lãi suất Ngân hàng Trung ương Thụy Sĩ (SNB)', time: '14:30 ngày mai', importance: 'high', source: 'Reuters' }
    ]
  },
  '2026-06-16': {
    date: '2026-06-16',
    quickHighlights: [
      'Thị trường chứng khoán Mỹ đi ngang trước kỳ nghỉ lễ, các nhà đầu tư chốt lời cổ phiếu chip sau chuỗi tăng điểm nóng.',
      'Giá dầu thế giới suy giảm nhẹ do lo ngại về tiến trình phục hồi nhu cầu tại Trung Quốc yếu hơn kỳ vọng bất chấp mùa hè cao điểm ở phương Tây.',
      'Đồng USD duy trì sức mạnh ổn định khi lạm phát khu vực đồng Euro có dấu hiệu cứng đầu, làm giảm triển vọng ECB sớm hạ lãi suất tiếp theo.'
    ],
    keyNumbers: [
      { label: 'S&P 500', value: '5,431.60', change: '-0.12%', isPositive: false, sparkline: [5445, 5450, 5442, 5438, 5430, 5435, 5429, 5431.60] },
      { label: 'Dầu Brent', value: '$84.25', change: '-0.45%', isPositive: false, sparkline: [84.8, 84.9, 84.6, 84.5, 84.3, 84.1, 84.3, 84.25] },
      { label: 'Vàng thế giới', value: '$2,322.60', change: '-0.38%', isPositive: false, sparkline: [2335, 2332, 2329, 2325, 2322, 2326, 2320, 2322.6] }
    ],
    stocks: {
      indexName: 'S&P 500 & Dow Jones Industrial',
      value: '5,431.60 / 38,778.14',
      change: '-6.52 / -115.80',
      changePercent: '-0.12% / -0.30%',
      isPositive: false,
      chartPoints: [5445, 5450, 5440, 5432, 5428, 5436, 5429, 5431.6],
      chartLabels: ['9:30', '10:30', '11:30', '12:30', '13:30', '14:30', '15:30', '16:00'],
      thumbnail: '/images/cnbc_trader.png',
      highlights: [
        'Thận trọng trước kỳ nghỉ: Thanh khoản sụt giảm khi phố Wall chuẩn bị bước vào ngày nghỉ lễ Juneteenth, khiến các vị thế mua mới hạn chế đáng kể.',
        'Áp lực điều chỉnh: Cổ phiếu công nghệ sinh học và y tế chịu lực bán mạnh, trong khi Apple và Microsoft giữ nhịp tăng nhẹ giúp thị trường không bị giảm sâu.',
        'Thị trường châu Á: Chỉ số Hang Seng của Hồng Kông giảm 1.3% do nhóm bất động sản tiếp tục chịu sức ép.'
      ],
      cnbcUrl: 'https://www.cnbc.com/world/',
      wsjUrl: 'https://www.wsj.com/finance/stocks?mod=nav_top_subsection',
      reutersUrl: 'https://www.reuters.com/markets/stocks/'
    },
    oil: {
      wtiPrice: '$80.44',
      wtiChange: '-$0.36',
      wtiPercent: '-0.45%',
      wtiPositive: false,
      brentPrice: '$84.25',
      brentChange: '-$0.38',
      brentPercent: '-0.45%',
      brentPositive: false,
      chartPointsWTI: [80.8, 80.9, 80.7, 80.5, 80.3, 80.2, 80.5, 80.44],
      chartPointsBrent: [84.6, 84.8, 84.5, 84.4, 84.1, 84.0, 84.3, 84.25],
      chartLabels: ['9:00', '11:00', '13:00', '15:00', '17:00', '19:00', '21:00', '23:00'],
      thumbnail: '/images/reuters_oil.png',
      highlights: [
        'Dữ liệu từ Trung Quốc: Sản lượng lọc dầu của Trung Quốc giảm nhẹ trong tháng trước, làm dấy lên nghi ngại về tốc độ hấp thụ dầu thô của quốc gia nhập khẩu dầu mỏ lớn nhất hành tinh này.',
        'Sàn NYMEX/ICE: Phiên chốt WTI rút chân nhẹ về cuối ngày giữ mốc $80, Brent duy trì trên $84 nhờ sự nâng đỡ từ rủi ro địa chính trị địa phương.',
        'Kế hoạch OPEC+: Các quan chức Ả Rập Xê Út nhắc lại rằng liên minh có thể linh hoạt đảo ngược quyết định tăng sản lượng vào tháng 10 nếu thị trường yếu.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/',
      nytimesUrl: 'https://www.nytimes.com/section/business/energy-environment?page=2',
      wsjUrl: 'https://www.wsj.com/business/energy-oil?mod=nav_top_subsection'
    },
    goldUsd: {
      goldPrice: '$2,322.60',
      goldChange: '-$8.90',
      goldPercent: '-0.38%',
      goldPositive: false,
      dxyPrice: '105.52',
      dxyChange: '+0.12',
      dxyPercent: '+0.11%',
      dxyPositive: true,
      chartPointsGold: [2334, 2330, 2328, 2322, 2320, 2324, 2321, 2322.6],
      chartPointsDXY: [105.40, 105.42, 105.45, 105.48, 105.55, 105.50, 105.53, 105.52],
      chartLabels: ['8:00', '10:00', '12:00', '14:00', '16:00', '18:00', '20:00', '22:00'],
      highlights: [
        'Vàng giảm giá: Đồng USD mạnh và lợi suất trái phiếu nhích lên đẩy giá vàng rời xa vùng kháng cự $2.340.',
        'Thông điệp từ Fed: Nhiều đại diện của Fed nhấn mạnh cần thấy thêm vài tháng lạm phát hạ nhiệt liên tiếp nữa trước khi nghĩ tới việc cắt giảm lãi suất điều hành.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/'
    },
    vtvIndexNews: [
      {
        title: 'Nền kinh tế Mỹ có nguy cơ đình lạm nhẹ vào cuối năm?',
        summary: 'Các nhà kinh tế học cảnh báo tình trạng chi tiêu tiêu dùng chậm lại đi kèm giá dịch vụ vẫn cao có thể đưa kinh tế Mỹ vào kịch bản đình lạm nhẹ.',
        source: 'VTVIndex Thế giới',
        time: '1 ngày trước'
      },
      {
        title: 'Tăng trưởng xanh: Châu Âu gấp rút tự chủ tấm quang năng',
        summary: 'Uỷ ban châu Âu đề xuất quỹ hỗ trợ trị giá 12 tỷ Euro cho các nhà máy lắp ráp pin mặt trời tại Đức và Ba Lan nhằm đối phó nguồn cung giá rẻ.',
        source: 'VTVIndex Thế giới',
        time: '1 ngày trước'
      }
    ],
    vietnamFinanceNews: [
      {
        title: 'Ngân hàng đẩy mạnh bán lẻ số để cải thiện tỷ lệ tiền gửi không kỳ hạn (CASA)',
        summary: 'VNeconomy đưa tin cuộc đua cải tiến ứng dụng di động ngân hàng đang trở nên gay gắt khi chi phí vốn huy động có kỳ hạn tăng nhẹ trở lại.',
        source: 'VNeconomy',
        time: '1 ngày trước'
      },
      {
        title: 'Thị trường bất động sản công nghiệp tiếp tục là điểm sáng thu hút FDI',
        summary: 'Nhịp cầu đầu tư dẫn số liệu cho thấy tỷ lệ lấp đầy các khu công nghiệp tại Bình Dương và Bắc Ninh đạt trung bình trên 88% nhờ làn sóng dịch chuyển sản xuất.',
        source: 'Tạp chí Nhịp cầu đầu tư',
        time: '1 ngày trước'
      }
    ],
    breakingNews: [
      { time: '17:30', source: 'Bloomberg', content: 'Chính phủ Pháp bác bỏ thông tin cho rằng nước này đang thảo luận các biện pháp kiểm soát dòng vốn để bảo vệ ngân hàng.', isUrgent: true },
      { time: '15:10', source: 'Reuters', content: 'Doanh số bán lẻ lõi của Trung Quốc trong tháng 5 chỉ tăng 3.7%, thấp hơn kỳ vọng đáng kể.', isUrgent: false }
    ],
    watchlist: [
      { event: 'Dữ liệu Bán lẻ Mỹ (Tháng 5)', time: '19:30', importance: 'high', source: 'Bloomberg' },
      { event: 'Tồn kho dầu thô thương mại EIA (Mỹ)', time: '21:30', importance: 'medium', source: 'Reuters' }
    ]
  },
  '2026-06-15': {
    date: '2026-06-15',
    quickHighlights: [
      'Thị trường tài chính khởi đầu tuần mới trong trạng thái chờ đợi khi thiếu vắng tin tức kinh tế quan trọng.',
      'Giá dầu Brent đi lên nhẹ vượt ngưỡng $84/thùng do kỳ vọng nhu cầu gia tăng đột biến vào kỳ nghỉ du lịch hè ở Bắc Mỹ.',
      'Giá vàng duy trì trong biên độ hẹp quanh $2.330 khi tâm lý e dè bao trùm thị trường sau đợt biến động mạnh tuần trước.'
    ],
    keyNumbers: [
      { label: 'S&P 500', value: '5,438.12', change: '+0.18%', isPositive: true, sparkline: [5420, 5425, 5430, 5428, 5432, 5435, 5436, 5438.12] },
      { label: 'Dầu Brent', value: '$84.63', change: '+0.75%', isPositive: true, sparkline: [83.9, 84.1, 84.0, 84.2, 84.3, 84.5, 84.4, 84.63] },
      { label: 'Vàng thế giới', value: '$2,331.50', change: '+0.12%', isPositive: true, sparkline: [2325, 2328, 2324, 2330, 2327, 2332, 2329, 2331.5] }
    ],
    stocks: {
      indexName: 'S&P 500 & Dow Jones Industrial',
      value: '5,438.12 / 38,894.20',
      change: '+9.75 / +88.30',
      changePercent: '+0.18% / +0.23%',
      isPositive: true,
      chartPoints: [5422, 5428, 5432, 5426, 5430, 5434, 5432, 5438.12],
      chartLabels: ['9:30', '10:30', '11:30', '12:30', '13:30', '14:30', '15:30', '16:00'],
      thumbnail: '/images/cnbc_trader.png',
      highlights: [
        'Khởi đầu tuần mới yên ả: Thị trường ít có biến động lớn do không có công bố chỉ số kinh tế lớn của Mỹ đầu tuần.',
        'Sự phục hồi nhẹ: Nhóm hàng tiêu dùng và năng lượng dẫn đầu sắc xanh nhẹ, trong khi nhóm công nghệ chững lại sau chuỗi ngày tăng nóng.',
        'Diễn biến châu Âu: Chỉ số CAC 40 của Pháp hồi phục nhẹ 0.9% sau khi giảm sâu vào tuần trước do bất ổn từ cuộc bầu cử sớm.'
      ],
      cnbcUrl: 'https://www.cnbc.com/world/',
      wsjUrl: 'https://www.wsj.com/finance/stocks?mod=nav_top_subsection',
      reutersUrl: 'https://www.reuters.com/markets/stocks/'
    },
    oil: {
      wtiPrice: '$80.80',
      wtiChange: '+$0.68',
      wtiPercent: '+0.85%',
      wtiPositive: true,
      brentPrice: '$84.63',
      brentChange: '+$0.63',
      brentPercent: '+0.75%',
      brentPositive: true,
      chartPointsWTI: [80.1, 80.3, 80.2, 80.5, 80.6, 80.4, 80.7, 80.8],
      chartPointsBrent: [83.9, 84.1, 84.0, 84.3, 84.4, 84.2, 84.5, 84.63],
      chartLabels: ['9:00', '11:00', '13:00', '15:00', '17:00', '19:00', '21:00', '23:00'],
      thumbnail: '/images/reuters_oil.png',
      highlights: [
        'Kỳ vọng nhu cầu mùa hè: Nhu cầu đi lại bằng đường hàng không và đường bộ tại Mỹ trong mùa hè được kỳ vọng sẽ tăng vọt, tiêu thụ lượng lớn xăng dầu.',
        'Địa chính trị bảo vệ giá: Căng thẳng tiếp tục diễn ra dọc các tuyến vận tải biển chính ở biển Đỏ ngăn cản đà bán tháo của dầu thô thương mại.',
        'Dự báo Goldman Sachs: Ngân hàng đầu tư này dự báo giá dầu Brent sẽ duy trì trong phạm vi $80 - $86/thùng suốt quý III năm nay.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/',
      nytimesUrl: 'https://www.nytimes.com/section/business/energy-environment?page=2',
      wsjUrl: 'https://www.wsj.com/business/energy-oil?mod=nav_top_subsection'
    },
    goldUsd: {
      goldPrice: '$2,331.50',
      goldChange: '+$2.80',
      goldPercent: '+0.12%',
      goldPositive: true,
      dxyPrice: '105.40',
      dxyChange: '-0.08',
      dxyPercent: '-0.08%',
      dxyPositive: false,
      chartPointsGold: [2328, 2330, 2326, 2332, 2329, 2334, 2330, 2331.5],
      chartPointsDXY: [105.48, 105.45, 105.42, 105.44, 105.38, 105.41, 105.39, 105.40],
      chartLabels: ['8:00', '10:00', '12:00', '14:00', '16:00', '18:00', '20:00', '22:00'],
      highlights: [
        'Giao dịch thận trọng: Giá vàng biến động nhẹ do nhà đầu tư vẫn đang tiêu hóa thông điệp mang tính "diều hâu" hơn dự đoán từ phiên họp FOMC tuần trước.',
        'Sự ổn định của DXY: Đồng USD đi ngang quanh mốc 105.4 khi thị trường chờ đợi dữ liệu vĩ mô mới để có thêm động lực giao dịch.'
      ],
      reutersUrl: 'https://www.reuters.com/markets/'
    },
    vtvIndexNews: [
      {
        title: 'Châu Á gia tăng vị thế trong chuỗi cung ứng linh kiện bán dẫn toàn cầu',
        summary: 'Các nhà đầu tư đổ dồn vốn vào trung tâm công nghệ mới ở Đông Nam Á, tiêu biểu là Việt Nam và Malaysia, thúc đẩy thị trường bất động sản công nghiệp phát triển mạnh.',
        source: 'VTVIndex Thế giới',
        time: '2 ngày trước'
      }
    ],
    vietnamFinanceNews: [
      {
        title: 'Vàng nhẫn trơn tiếp tục khan hiếm tại các cửa hàng lớn ở Hà Nội',
        summary: 'Kinh tế Sài Gòn ghi nhận tình trạng người dân xếp hàng mua vàng nhẫn vẫn diễn ra do doanh nghiệp hạn chế bán lẻ lượng lớn để bảo toàn nguồn hàng nguyên liệu.',
        source: 'Tạp chí Kinh tế Sài Gòn',
        time: '2 ngày trước'
      }
    ],
    breakingNews: [
      { time: '11:20', source: 'CNBC', content: 'Chính phủ Thụy Điển tuyên bố sẽ tiếp tục hạ thuế thu nhập để khuyến khích thị trường lao động phát triển.', isUrgent: false },
      { time: '09:05', source: 'Reuters', content: 'Ngân hàng Trung ương Trung Quốc giữ nguyên lãi suất cho vay trung hạn (MLF) kỳ hạn 1 năm ở mức 2.50%.', isUrgent: true }
    ],
    watchlist: [
      { event: 'Phát biểu của Thống đốc RBA (Úc)', time: '10:30', importance: 'medium', source: 'Reuters' },
      { event: 'Chỉ số lạm phát lõi CPI của Eurozone', time: '16:00', importance: 'high', source: 'Financial Times' }
    ]
  }
};
