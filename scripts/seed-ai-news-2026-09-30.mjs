const API = process.env.PAGES_API || "http://localhost:8080";
const TOKEN = process.env.PAGE_TOKEN;
if (!TOKEN) throw new Error("PAGE_TOKEN is required");

const news = [
  ["openai-devday-2026", "OpenAI DevDay 2026：智能体成为新工作入口", "OpenAI", "2026-09-29", "智能体", "OpenAI 发布超过 20 项更新，把持续运行的智能体、Codex、插件与团队协作整合为统一工作面。", ["Dots 被定义为可持续承担职责的常驻智能体", "Agents API 加入 computer use 与多智能体能力", "ChatGPT Space、Pages 与团队任务强化人机协作", "开发者可在 ChatGPT 内发布原生插件体验"], ["评估智能体是否能接管端到端流程，而不只是生成内容", "提前建立工具权限、审计日志和人工接管机制", "选择一个高频、可量化的流程开展两周试点"], "https://openai.com/index/devday-2026-recap/"],
  ["openai-gpt-6-1-sol", "GPT-6.1 Sol：更低成本的高性能智能体模型", "OpenAI", "2026-09-29", "模型", "GPT-6.1 Sol 聚焦智能体编码、计算机使用与专业工作，以更低成本逼近旗舰模型能力。", ["官方称标准输入输出价格约为 Astra 的五分之一", "DeepSWE、AutomationBench 与 OSWorld 等评测均有提升", "缓存输入降至每百万 token 0.10 美元", "面向 API、Codex 与 ChatGPT Work 提供"], ["重新测算复杂智能体任务的单位成功成本", "按任务难度建立 Sol 与旗舰模型的路由策略", "在真实业务数据上复测事实性和工具调用稳定性"], "https://openai.com/index/introducing-gpt-6-1-sol/"],
  ["openai-dots", "OpenAI Dots：从对话助手走向常驻代理", "OpenAI", "2026-09-29", "智能体", "Dots 主打持续理解用户目标、主动推进工作与长期承担职责，代表 AI 产品从会话式工具转向常驻工作伙伴。", ["围绕长期目标而非单轮提示组织任务", "强调持续工作、上下文积累与主动提醒", "优先面向 Pro 与 Business Premium 市场", "企业版本默认关闭，需要管理员启用"], ["定义代理可以自主执行和必须确认的边界", "把长期记忆拆分为偏好、项目事实与敏感数据", "为异常行为准备暂停、回滚和责任追踪机制"], "https://openai.com/index/introducing-dots/"],
  ["openai-frontier-safety-cases", "OpenAI 提出前沿模型训练安全论证框架", "OpenAI", "2026-09-28", "安全", "OpenAI 探索用结构化 safety case 证明前沿训练活动的风险处于可接受范围，并让关键判断可审查。", ["把风险主张、证据和假设组织成可检查链条", "关注训练前、中、后的控制措施", "强调证据随模型能力变化持续更新", "为外部评估和监管沟通提供共同语言"], ["将高风险 AI 项目的安全判断文档化", "把模型评测、红队和上线门槛连接到同一证据库", "明确谁对残余风险签字负责"], "https://openai.com/index/towards-safety-cases-for-frontier-ai-training/"],
  ["anthropic-sonnet-5-5", "Claude Sonnet 5.5：速度、成本与能力再平衡", "Anthropic", "2026-09-28", "模型", "Anthropic 将 Sonnet 5.5 定位为 Sonnet 5 的明确升级，强调更快推理和更低运行成本。", ["官方称多数工作速度提升约 30%", "多数工作成本最高可降低约 30%", "继续面向编码与知识工作主流场景", "释放模型组合重新分层的空间"], ["用内部黄金任务集比较质量、时延与成本", "检查长上下文和工具调用场景是否同样受益", "先灰度替换中等复杂度生产流量"], "https://www.anthropic.com/claude-sonnet-5-5"],
  ["anthropic-opus-5-5", "Claude Opus 5.5：旗舰能力开始下沉", "Anthropic", "2026-09-22", "模型", "Opus 5.5 以更低成本提供接近此前更高阶模型的表现，进一步压缩旗舰能力与日常生产模型之间的差距。", ["官方称多数任务达到 Claude Fable 5.1 水平", "相较 Opus 5 运行成本降低约 40%", "适合高价值分析、复杂编码和代理任务", "模型选型重点转向单位任务价值"], ["把高难任务从固定模型改为动态升级", "用完整任务成功率代替单次调用价格", "保留对关键结论的人工复核"], "https://www.anthropic.com/claude-opus-5-5"],
  ["anthropic-fable-mythos-5-1", "Claude Fable 5.1 与 Mythos 5.1：编码和科研双线推进", "Anthropic", "2026-09-01", "模型", "Anthropic 同时推出面向编码、知识工作与科研探索的新一代模型，展示通用能力和专业科学能力的并行演进。", ["强调复杂编码与知识工作", "研究能力被用于展示 AI 参与科学发现的潜力", "产品线出现更清晰的任务分工", "专业访问与安全控制同步增强"], ["按通用生产与专业科研建立不同评测集", "专业领域必须配置专家复核", "关注模型能力提升带来的双重用途风险"], "https://www.anthropic.com/claude-fable-and-mythos-5-1"],
  ["anthropic-threat-report-sep-2026", "Anthropic 九月威胁报告：AI 滥用正在专业化", "Anthropic", "2026-09-10", "安全", "Anthropic 汇总过去八个月中被发现并阻断的恶意使用案例，展示攻击者如何把模型嵌入更成熟的行动链。", ["威胁主体尝试用 Claude 支持恶意活动", "报告以案例说明滥用方式的演化", "防护重点从提示拦截扩展到行为模式", "平台情报需要跨产品和时间关联"], ["对高风险工具调用启用分级审批", "监控异常批量、自动化和身份切换行为", "将模型安全事件纳入现有 SOC 流程"], "https://www.anthropic.com/threat-intelligence-report-september-2026"],
  ["anthropic-claude-enzyme-discovery", "Claude 发现类 CRISPR 重复序列的新酶系统", "Anthropic", "2026-09-23", "科学 AI", "Anthropic 披露 Claude 智能体从大规模 DNA 数据中识别出此前未表征的酶系统，并由人类科学家开展实验验证。", ["约 950 个代理运行 21 小时并消耗 2.1 亿 token", "从逾 20 万个逆转录酶缩小到 20 个候选", "系统包含类似 CRISPR 的 DNA 重复阵列", "发现仍处早期阶段，具体功能有待实验确认"], ["把 AI 用于扩大假设搜索空间而非替代实验", "保留数据、推理轨迹和候选淘汰依据", "对生物安全和结论外推设置严格边界"], "https://www.anthropic.com/news/claude-discovers-novel-enzyme-system"],
  ["anthropic-accenture-evaluation", "Anthropic 与 Accenture 试点嵌入式独立评估", "Anthropic", "2026-09-18", "治理", "双方计划让独立评估人员更早进入模型开发流程，开展红队、对齐评估和防护验证。", ["评估者可获得接近内部员工的开发可见度", "双方预计五年内各投入至少 10 亿美元建设能力", "合作非排他，未来将引入更多评估机构", "评估标准、信息边界与资金机制仍待完善"], ["采购模型时要求可验证的第三方评估证据", "区分供应商自测、外部评测与监管审查", "关注评估机构的独立性和利益冲突"], "https://www.anthropic.com/news/accenture-embedded-evaluation"],
  ["anthropic-life-sciences-verification", "Anthropic 推出生物科学验证访问计划", "Anthropic", "2026-09-17", "科学 AI", "Life Sciences Verification Program 为经验证的生命科学团队提供更适配专业研究的模型访问与安全策略。", ["覆盖药物发现、研究生物学、临床开发和制造", "面向团队和机构以 beta 形式开放", "访问 Mythos、Opus 与 Sonnet 模型", "通过身份和用途验证平衡能力开放与风险"], ["专业能力开放应与组织验证、用途声明绑定", "建立数据合规、实验安全和成果发布审查", "持续记录模型对研究决策的实际影响"], "https://www.anthropic.com/news/life-sciences-verification-program"],
  ["anthropic-enterprise-frontier-safeguards", "Anthropic 与企业客户共建前沿模型防护", "Anthropic", "2026-09-01", "企业 AI", "Anthropic 将企业真实部署经验纳入前沿模型防护设计，推动安全控制从通用政策走向场景化治理。", ["企业场景暴露更复杂的数据与权限边界", "防护需要覆盖模型、工具、身份和流程", "客户反馈可帮助识别实验室评测遗漏", "治理能力成为企业采用前沿模型的前提"], ["建立企业 AI 使用场景和风险分级目录", "把最小权限、数据隔离和审计设为默认项", "对高影响流程开展上线前演练"], "https://www.anthropic.com/news/enterprise-frontier-safeguards"],
  ["google-gemini-3-8-flash-cyber", "Gemini 3.8 Flash 与 Flash Cyber：智能体和安全专用化", "Google", "2026-09-02", "模型", "Google 发布 Gemini 3.8 Flash 及面向可信防御者的 Cyber 版本，强化长周期编码、代理推理和漏洞发现。", ["Flash 保持与 3.7 相近的速度和定价", "官方定价为每百万输入 0.75 美元、输出 3.75 美元", "Cyber 版本聚焦漏洞检测和自动修复", "高风险安全能力通过 Fairwind 计划受控开放"], ["分别评估通用代理和安全代理，避免混用基准", "对自动修复启用沙箱、变更评审和回滚", "按推理强度调节 token 成本"], "https://blog.google/innovation-and-ai/models-and-research/gemini-models/3-8-flash-and-3-8-flash-cyber/"],
  ["google-gemini-live-avatar", "Gemini 3.8 Live Avatar：实时交互进入可视化阶段", "Google", "2026-09", "多模态", "Gemini 3.8 Live 将实时语音交互扩展到动态 Avatar，使数字角色可以结合对话状态进行更自然的表达。", ["实时交互从声音扩展到视觉形象", "适合教育、客服、陪伴和虚拟主持场景", "体验质量取决于低延迟与跨模态一致性", "身份呈现和深度伪造风险同步上升"], ["明确向用户披露 AI 身份", "对形象、声音授权和内容留痕建立规范", "在弱网和打断场景下测试交互质量"], "https://blog.google/innovation-and-ai/models-and-research/gemini-models/gemini-3-8-live-with-live-avatar/"],
  ["google-private-ai-compute-memory", "Google Private AI Compute 引入安全服务端记忆", "Google DeepMind", "2026-09", "隐私", "Google 推进 Private AI Compute 的安全服务端记忆，让个性化体验与敏感数据保护尝试兼得。", ["服务端记忆可支持跨会话个性化", "隐私计算用于减少基础设施对明文数据的暴露", "记忆生命周期与删除能力成为关键", "技术保证仍需配合清晰的产品控制"], ["让用户查看、纠正和删除长期记忆", "按敏感度设置不同保留期限", "对隐私计算声明进行独立验证"], "https://deepmind.google/blog/advancing-private-ai-compute-with-secure-server-side-memory/"],
  ["google-gemini-3-8-tts", "Gemini 3.8 TTS：语音生成向可控表达升级", "Google", "2026-09", "语音", "Gemini 3.8 Flash TTS 与 Flash-Lite TTS 面向自然语音应用，强调表达控制、低延迟与规模化部署。", ["双型号覆盖质量优先与效率优先场景", "语音不再只是朗读，而是可编排的表达层", "适合实时助手、内容制作和无障碍产品", "声音身份和合成内容标识不可忽视"], ["用真实脚本测试情绪、数字和专有名词", "建立声音授权与合成标识机制", "将首包延迟和打断恢复纳入验收"], "https://blog.google/innovation-and-ai/models-and-research/gemini-models/gemini-3-8-text-to-speech/"],
  ["google-gemini-live-thinking", "Gemini 3.8 Live Extended Thinking：实时与深度推理结合", "Google", "2026-09", "多模态", "Gemini 3.8 Live 系列尝试在实时对话中加入扩展思考能力，使复杂问题处理不再局限于短响应。", ["实时交互与长链推理开始融合", "系统需要在响应速度和推理质量间动态平衡", "适合辅导、诊断辅助和复杂客服", "更长推理会放大成本与等待体验问题"], ["按问题复杂度自动切换思考预算", "向用户呈现等待状态和可取消操作", "对高风险建议加入依据与复核入口"], "https://blog.google/innovation-and-ai/models-and-research/gemini-models/gemini-3-8-live-gemini-3-8-live-extended-thinking/"],
  ["google-alphagenome-atlas", "AlphaGenome Atlas：绘制 90 亿人类 DNA 变异预测图谱", "Google DeepMind", "2026-09", "科学 AI", "AlphaGenome Atlas 将大规模变异影响预测组织为可查询图谱，为遗传研究和疾病机制探索提供新工具。", ["覆盖约 90 亿种可能的单字母 DNA 变化", "帮助研究者优先筛选值得实验验证的变异", "从单次模型调用走向基础科研数据产品", "预测结果不等同于临床诊断"], ["把模型预测作为候选排序而非最终证据", "记录数据版本、模型版本和不确定性", "临床使用前必须经过独立验证与监管评估"], "https://deepmind.google/blog/alphagenome-atlas-a-predictive-map-of-every-possible-dna-letter-change-in-the-human-genome/"],
  ["google-weathernext-3", "WeatherNext 3：AI 天气模型继续逼近业务核心", "Google", "2026-09", "科学 AI", "WeatherNext 3 被 Google 定位为其更先进、准确的全球天气 AI 模型，推动快速概率预报进入更多实际决策。", ["AI 模型持续提升全球预测能力", "更快生成可支持多情景概率分析", "能源、物流、农业和灾害响应均可受益", "极端事件仍需要传统模型与专家交叉验证"], ["从一个具体决策场景评估预测价值", "比较准确率之外的提前量和校准度", "为错误预报设置业务缓冲与人工复核"], "https://blog.google/innovation-and-ai/models-and-research/google-deepmind/introducing-weathernext-3/"],
  ["google-fairwind-cyber-defense", "Google Fairwind：高能力网络安全 AI 采用受控开放", "Google", "2026-09", "安全", "Fairwind Program 向可信合作伙伴提供更强的网络防御模型与工具，探索双重用途能力的分级访问。", ["高能力安全模型不直接全面开放", "合作伙伴资格与用途限制构成第一道防线", "重点能力包括漏洞发现和防御自动化", "计划体现能力开放与滥用防范的权衡"], ["对安全代理实行实名、最小权限和隔离环境", "记录每次扫描、利用验证和修复动作", "建立漏洞披露与紧急停用流程"], "https://blog.google/innovation-and-ai/technology/safety-security/fairwind-program/"],
];

const esc = (s) => String(s).replace(/[&<>\"]/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c]));
const list = (xs) => xs.map(x => `<li>${esc(x)}</li>`).join("");
const guides = {
  "模型": {
    context:"基础模型市场已经从单纯追逐参数和榜单，进入能力、成本、速度、上下文、工具使用与可靠性综合竞争阶段。新版本的真正价值，需要放在完整业务任务中衡量。",
    opportunities:["降低复杂推理和长流程自动化的单位成本","用模型路由覆盖从日常任务到专家任务的不同需求","提升代码、文档和跨工具工作的自动化比例"],
    risks:["厂商基准与真实业务分布可能存在明显差异","版本升级可能改变输出风格、延迟和安全边界","更强的自主能力会放大错误操作的影响范围"],
    metrics:["端到端任务成功率","每个成功任务总成本","P50 / P95 响应时延","事实错误与人工接管率"]
  },
  "智能体": {
    context:"智能体正在从一次性问答转向持续感知目标、调用工具、保存状态并主动推进工作的系统。竞争焦点因此从生成质量扩展到执行可靠性与治理能力。",
    opportunities:["承接跨系统、长周期的重复业务流程","把团队知识和操作规范转成可复用执行能力","让人类从操作步骤转向目标设定和异常决策"],
    risks:["权限过大可能导致越权访问或不可逆操作","长期记忆可能累积错误、过期信息和敏感数据","多步骤执行中的小错误可能层层放大"],
    metrics:["无人干预完成率","平均人工接管次数","工具调用失败率","任务回滚率与审计覆盖率"]
  },
  "安全": {
    context:"模型能力增强后，安全不再只是内容过滤，而是覆盖身份、权限、工具、运行环境、监控、事件响应和外部审查的完整系统工程。",
    opportunities:["用 AI 扩大漏洞发现、告警研判和修复能力","建立可验证的模型上线门槛与持续评估体系","将安全证据转化为客户信任和采购优势"],
    risks:["网络安全能力具有明显双重用途","静态红队结果难以覆盖持续变化的攻击方式","缺少独立审查会导致安全声明难以验证"],
    metrics:["高风险行为拦截率","误报与漏报率","事件发现和处置时间","高风险操作审批覆盖率"]
  },
  "科学 AI": {
    context:"AI 正从阅读论文和辅助分析，走向大规模生成假设、检索候选、运行计算实验并与湿实验形成闭环。速度提升必须与可复现性、领域审查和安全控制同步。",
    opportunities:["在海量候选中更快发现值得验证的方向","自动化文献、数据、代码和实验记录之间的连接","缩短从假设到实验设计的准备周期"],
    risks:["模型预测不等于实验事实或临床证据","训练数据偏差可能形成系统性研究盲点","生物、医疗等领域存在高影响双重用途风险"],
    metrics:["有效候选命中率","实验复现率","专家复核通过率","从假设到验证的周期与成本"]
  },
  "治理": {
    context:"前沿 AI 治理正在从原则声明走向可检查的评估、证据、责任和报告机制。关键问题是评估者能否获得充分信息，并保持真正独立。",
    opportunities:["把安全和合规要求前置到模型开发阶段","建立供应商、客户和监管者可共享的证据框架","用持续评估替代一次性上线审批"],
    risks:["评估机构的资金关系可能影响独立性","商业机密与透明披露之间存在现实冲突","缺少统一标准会造成结果不可比较"],
    metrics:["独立评估覆盖率","重大问题整改闭环率","安全承诺可验证比例","外部报告及时性"]
  },
  "企业 AI": {
    context:"企业 AI 已进入规模化阶段，价值不只来自模型本身，还取决于数据接入、身份权限、业务流程、运营支持和组织变革是否协同。",
    opportunities:["把企业知识转化为可执行的工作流","提升跨部门信息流转和决策效率","形成行业专属的评测、知识和流程资产"],
    risks:["数据边界不清会引发泄露和合规问题","过度自动化可能削弱责任归属和人工判断","供应商锁定会增加迁移和长期成本"],
    metrics:["活跃使用率与复用率","业务周期缩短比例","人工节省时长","合规事件与单位价值成本"]
  },
  "多模态": {
    context:"实时语音、视觉与深度推理正在合并为统一交互界面。产品体验的上限由模型能力决定，下限则由延迟、打断处理、身份透明和跨模态一致性决定。",
    opportunities:["打造更自然的教育、客服和辅助交互","降低复杂软件和专业知识的使用门槛","形成可理解环境并实时行动的新型助手"],
    risks:["视觉和声音拟真度会加剧身份欺骗风险","弱网、噪声和多人场景可能快速拉低可靠性","实时采集带来更复杂的隐私和同意问题"],
    metrics:["首响应与端到端延迟","打断恢复成功率","跨模态事实一致率","用户任务完成率"]
  },
  "隐私": {
    context:"长期记忆让 AI 更有用，也使数据生命周期、访问边界和可删除性成为产品核心。隐私计算提供技术基础，但不能代替清晰的用户控制与治理。",
    opportunities:["在保护敏感数据的前提下提供持续个性化","减少重复输入并提升长期任务连续性","用可验证隐私能力进入高合规行业"],
    risks:["错误记忆可能长期影响后续决策","用户难以理解数据实际存储和处理位置","删除、导出和纠正机制若不完整会损害信任"],
    metrics:["记忆纠正与删除成功率","敏感信息误存率","隐私预算与访问审计覆盖率","用户控制功能使用率"]
  },
  "语音": {
    context:"语音模型正从机械朗读升级为可控制情绪、节奏、角色和多语言表达的生成层，并逐步进入实时对话和大规模内容生产。",
    opportunities:["提升无障碍、客服、教育和内容本地化体验","把品牌声音转化为可规模化的数字资产","用自然交互降低复杂产品的学习成本"],
    risks:["未经授权的声音克隆会形成欺诈风险","语气自然并不代表内容正确","多语言与方言质量可能存在明显不均衡"],
    metrics:["首包音频延迟","专有名词准确率","主观自然度评分","合成标识与授权覆盖率"]
  }
};
const themes = [
  ["#287a65","#e7f6ef","#f7fbf8","#d5efe4"], ["#316c91","#e8f4fa","#f8fbfd","#d7ebf5"],
  ["#7a5aa6","#f1ebf8","#fbf9fd","#e8ddf5"], ["#b76545","#fff0e7","#fffaf6","#f8dfd1"],
  ["#5b7b31","#eff7df","#fbfdf7","#e3f0cb"]
];
const impact = (category) => ({
  "模型":"模型供应进一步商品化，组织的差异化将更多来自评测体系、数据与流程设计。",
  "智能体":"软件界面正从功能菜单转向目标委托，权限系统和可观测性会成为新的基础设施。",
  "安全":"高能力 AI 同时提升攻防两端，可信访问、隔离执行和持续监控缺一不可。",
  "科学 AI":"科研瓶颈将从候选生成逐渐转向实验验证、数据质量和跨学科判断。",
  "治理":"治理竞争开始进入证据化阶段，无法被审查的安全承诺将越来越缺乏说服力。",
  "企业 AI":"企业竞争优势不在于是否接入模型，而在于能否安全地重构端到端工作。",
  "多模态":"实时、多模态交互将成为新的产品入口，也会重塑内容真实性与用户信任。",
  "隐私":"记忆能力越强，用户对数据透明、控制和可撤回性的要求就越高。",
  "语音":"语音将从附属功能变成主要交互层，品牌、版权和身份治理会同步升级。"
}[category] || "这项进展将持续影响 AI 产品、组织流程与治理方式。 ");
function page(n, i) {
  const [slug,title,source,date,category,summary,signals,actions,url] = n;
  const guide = guides[category] || guides["企业 AI"];
  const [accent,soft,paper,blob] = themes[i % themes.length];
  const read = 8 + (i % 4);
  const confidence = date.length === 10 ? "已确认发布日期" : "月度发布";
  return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${esc(title)}</title><style>
  :root{--accent:${accent};--soft:${soft};--paper:${paper};--blob:${blob};--ink:#17312a;--muted:#667a73;--line:#dbe8e2}*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:var(--paper);color:var(--ink);font:16px/1.8 ui-sans-serif,system-ui,-apple-system,"PingFang SC",sans-serif}.shell{max-width:1180px;margin:auto;padding:24px}.nav{display:flex;justify-content:space-between;align-items:center;padding:10px 0 28px}.brand{font-weight:900;color:var(--accent);letter-spacing:.15em}.nav a{color:var(--muted);text-decoration:none}.hero{position:relative;overflow:hidden;background:linear-gradient(135deg,var(--soft),#fff);border:1px solid var(--line);border-radius:34px;padding:64px clamp(28px,7vw,82px);box-shadow:0 20px 70px #31584812}.hero:after{content:"";position:absolute;width:360px;height:360px;border-radius:47% 53% 68% 32%;background:var(--blob);right:-110px;top:-150px;transform:rotate(${i*17}deg)}.hero>*{position:relative;z-index:1}.eyebrow,.kicker{color:var(--accent);font-weight:850;font-size:12px;letter-spacing:.17em;text-transform:uppercase}.hero h1{font:700 clamp(38px,6vw,72px)/1.1 Georgia,"Songti SC",serif;max-width:900px;margin:18px 0}.lead{font-size:20px;max-width:800px;color:#47635a}.meta,.tags{display:flex;gap:10px;flex-wrap:wrap;margin-top:24px}.pill{background:#ffffffaa;border:1px solid var(--line);border-radius:999px;padding:7px 13px;color:var(--muted);font-size:14px}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin:24px 0}.stat{background:#fff;border:1px solid var(--line);border-radius:20px;padding:20px}.stat b{display:block;font:700 25px Georgia,serif;color:var(--accent)}.layout{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(280px,.65fr);gap:24px}.card{background:#fff;border:1px solid var(--line);border-radius:24px;padding:30px;margin-bottom:24px;box-shadow:0 12px 35px #3158480b}.card.tint{background:var(--soft)}.card h2{font:700 27px Georgia,"Songti SC",serif;margin:6px 0 18px}.card h3{margin:22px 0 6px}.card li{margin:10px 0}.fact-list{list-style:none;padding:0;counter-reset:f}.fact-list li{counter-increment:f;display:grid;grid-template-columns:38px 1fr;gap:12px;align-items:start}.fact-list li:before{content:counter(f,decimal-leading-zero);color:var(--accent);font-weight:900}.quote{font:700 26px/1.55 Georgia,"Songti SC",serif;color:var(--accent)}.two{display:grid;grid-template-columns:1fr 1fr;gap:18px}.mini{border-top:3px solid var(--accent);background:var(--paper);border-radius:14px;padding:18px}.check{list-style:none;padding:0}.check li:before{content:"✓";color:var(--accent);font-weight:900;margin-right:10px}.watch{display:grid;grid-template-columns:1fr 1fr;gap:10px}.metric{background:var(--soft);border-radius:14px;padding:14px;font-size:14px}.source{display:block;color:var(--accent);text-decoration:none;word-break:break-all;font-weight:700}.source:hover{text-decoration:underline}.note{font-size:13px;color:var(--muted)}footer{margin:36px 0 20px;padding:25px 4px;border-top:1px solid var(--line);color:var(--muted)}@media(max-width:820px){.layout,.summary-grid,.two{grid-template-columns:1fr}.watch{grid-template-columns:1fr 1fr}.hero{padding:42px 26px}.shell{padding:14px}.hero h1{font-size:40px}}@media(max-width:460px){.watch{grid-template-columns:1fr}}
  </style></head><body><div class="shell"><nav class="nav"><span class="brand">AI SIGNAL</span><a href="#source">来源与边界 ↓</a></nav><header class="hero"><div class="eyebrow">${esc(category)} · 深度简报 · NO. ${String(i+1).padStart(2,"0")}</div><h1>${esc(title)}</h1><p class="lead">${esc(summary)}</p><div class="meta"><span class="pill">发布方：${esc(source)}</span><span class="pill">${esc(date)}</span><span class="pill">${read} 分钟深度阅读</span><span class="pill">${confidence}</span></div></header><section class="summary-grid"><div class="stat"><b>30 秒</b>先读结论与核心事实</div><div class="stat"><b>${signals.length} 个</b>已提炼的关键信号</div><div class="stat"><b>${actions.length} 步</b>可立即执行的建议</div></section><main class="layout"><div><article class="card tint"><div class="kicker">Executive brief</div><h2>一句话判断</h2><p class="quote">${esc(impact(category))}</p><p>${esc(summary)} 对团队而言，重点不是立即追随发布热度，而是验证它能否在真实流程中带来稳定、可衡量且风险可控的改进。</p></article><article class="card"><div class="kicker">Context</div><h2>背景与发展脉络</h2><p>${esc(guide.context)}</p><p>本次动态由 ${esc(source)} 于 ${esc(date)} 对外发布。它位于“能力快速提升”与“生产级落地要求提高”的交叉点：一方面，新能力扩大了可自动化的任务范围；另一方面，组织需要更严谨地处理成本、数据、权限、可靠性和责任归属。</p></article><article class="card"><div class="kicker">Verified facts</div><h2>核心事实与关键信号</h2><ol class="fact-list">${list(signals)}</ol><p class="note">以上为对官方材料的中文归纳；数字、可用范围和产品条款可能更新，应以原始页面为准。</p></article><article class="card"><div class="kicker">Analysis</div><h2>影响分析</h2><div class="two"><div class="mini"><h3>短期影响</h3><p>团队会优先开展能力验证、供应商比较和小规模试点。现有产品可能快速加入相似能力，采购与技术选型的比较维度将增多。</p></div><div class="mini"><h3>中长期影响</h3><p>${esc(impact(category))} 能形成持续优势的组织，会把评测、数据反馈、权限治理和流程改造沉淀为内部能力。</p></div></div><h3>潜在机会</h3><ul>${list(guide.opportunities)}</ul><h3>主要风险与限制</h3><ul>${list(guide.risks)}</ul></article><article class="card"><div class="kicker">Decision framework</div><h2>是否值得现在采用？</h2><div class="two"><div><h3>适合立即试点</h3><ul><li>已有清晰任务、基线数据和业务负责人</li><li>错误可发现、可回滚，影响范围可隔离</li><li>能够取得合法数据并完成安全评审</li></ul></div><div><h3>建议继续观察</h3><ul><li>需求仍停留在“想用 AI”而非具体问题</li><li>输出错误会直接造成高影响后果</li><li>缺少监控、审计、人工接管或退出方案</li></ul></div></div></article></div><aside><article class="card tint"><div class="kicker">Action plan</div><h2>团队行动建议</h2><ol>${list(actions)}</ol><h3>建议补充的治理动作</h3><ul class="check"><li>指定业务与技术双负责人</li><li>记录模型、提示、工具和数据版本</li><li>设置预算、权限和停止条件</li><li>安排上线后复盘节奏</li></ul></article><article class="card"><div class="kicker">Measurement</div><h2>持续观察指标</h2><div class="watch">${guide.metrics.map(x=>`<div class="metric">${esc(x)}</div>`).join("")}</div><p class="note">建议同时保留上线前基线，至少连续观察两个业务周期。</p></article><article class="card"><div class="kicker">Questions</div><h2>评审会上应追问</h2><ul><li>官方指标能否映射到我们的真实任务？</li><li>失败时谁能发现、停止并恢复？</li><li>数据会流向哪里，保留多久？</li><li>切换供应商或模型的成本是多少？</li><li>最终责任由谁承担并签字？</li></ul></article><article class="card" id="source"><div class="kicker">Source & scope</div><h2>来源与信息边界</h2><p><b>${esc(source)}</b><br>${esc(date)} · 官方发布</p><a class="source" href="${esc(url)}" target="_blank" rel="noopener noreferrer">阅读原始资料 ↗</a><p class="note">本页是基于公开官方资料制作的中文研究简报，包含编辑归纳与趋势判断。它不构成投资、医疗、法律、网络攻击或其他专业操作建议。涉及价格、可用地区、性能和政策时，请回到原文复核最新版本。</p></article></aside></main><footer>AI 最新资讯 · Information Radar<br>研究版 v2 · 更新于 2026-09-30 · 保留来源、结论与风险边界</footer></div></body></html>`;
}

async function request(path, options = {}) {
  const res = await fetch(`${API}${path}`, { ...options, headers: {Authorization:`Bearer ${TOKEN}`, ...(options.headers || {})} });
  const body = await res.text();
  if (!res.ok) throw new Error(`${res.status} ${path}: ${body}`);
  return body ? JSON.parse(body) : null;
}

let published = 0;
for (const [i, item] of news.entries()) {
  const [slug, title] = item;
  const uploaded = await request("/api/workspaces/information-radar/projects/ai/pages", {
    method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({slug,title,html:page(item,i)})
  });
  await request(`/api/workspaces/information-radar/projects/ai/pages/${slug}/publish`, {
    method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({version:uploaded.latestVersion})
  });
  published++;
  console.log(`${String(published).padStart(2,"0")}/20 ${slug}`);
}
console.log(JSON.stringify({workspace:"information-radar",project:"ai",published}));
