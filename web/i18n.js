(() => {
  const STORAGE_KEY = 'pages-language';
  const language = localStorage.getItem(STORAGE_KEY) === 'en' ? 'en' : 'zh-CN';
  const dictionary = {
    '为团队和 AI Agent 设计的页面发布平台':'A publishing platform for teams and AI agents',
    '从 AI 生成到团队协作，从安全预览到正式发布。':'From AI generation to team collaboration, from safe previews to production.',
    '让每一个想法，都成为可以访问的作品。':'Turn every idea into something people can visit.',
    '登录你的团队空间，继续发布精彩页面。':'Sign in to your team workspace and keep publishing.',
    '第一次使用 Pages，请创建管理员账号。':'Create the first administrator account to get started.',
    '安全登录 · 空间隔离 · 为人与 AI 协作而生':'Secure sign-in · Isolated workspaces · Built for people and AI',
    '拖拽调整空间顺序，管理空间名称与生命周期。空间标识创建后保持不变。':'Drag to reorder workspaces and manage their names and lifecycle. Workspace slugs remain stable.',
    '创建自己的工作空间，并管理你拥有的空间、项目和页面。':'Create workspaces and manage the spaces, projects, and pages you own.',
    '每次更新都安全、可控、可回退':'Every update is safe, controlled, and reversible',
    'Pages 为人工与 AI 生成的页面提供安全预览、正式发布和版本回滚。':'Pages provides safe previews, controlled publishing, and rollback for human- and AI-created pages.',
    '按产品、客户或主题组织页面，让团队资产保持清晰。':'Organize pages by product, customer, or topic to keep team assets clear.',
    '上传 AI 生成的 HTML，先在隔离环境中检查效果。':'Upload AI-generated HTML and review it in an isolated environment.',
    '确认后切换线上版本，需要时可随时回滚历史版本。':'Publish after review and roll back to any previous version when needed.',
    '在“空间设置 → API Token”创建发布密钥':'Create a publishing credential under Workspace Settings → API Token',
    '把工作空间与项目标识提供给 AI Agent':'Give the workspace and project slugs to your AI agent',
    '调用上传接口生成预览，再确认正式发布':'Use the upload API to create a preview, then publish it',
    '在工作空间中创建项目，然后发布 AI 生成的页面。':'Create a project in a workspace, then publish AI-generated pages.',
    '在工作空间中创建项目，然后发布 AI 生成的页面。':'Create a project in a workspace, then publish AI-generated pages.',
    '发布和管理这个项目中的页面。':'Publish and manage pages in this project.',
    '创建或加入工作空间，然后开始发布页面。':'Create or join a workspace, then start publishing pages.',
    '让每个想法，都有一个可访问的页面。':'Give every idea a page people can visit.',
    '创建工作空间和项目，通过安全预览把 AI 生成内容正式发布。':'Create workspaces and projects, then publish AI-generated content through safe previews.',
    '还没有可访问的工作空间':'No accessible workspaces yet',
    '创建工作空间后即可获得 API 接入地址':'Create a workspace to get its API endpoint',
    '还没有项目，从创建第一个项目开始。':'No projects yet. Create your first project to begin.',
    '上传后先预览，确认后再正式发布':'Preview the upload before publishing it',
    '相同标识会创建新版本':'The same slug creates a new version',
    '拖入 AI 生成的 HTML':'Drop AI-generated HTML here',
    '密钥明文只在创建或重置时显示一次':'The plaintext secret is shown only once when created or reset',
    '出于安全原因，平台不保存可找回的密钥明文。若密钥丢失，请执行重置。':'For security, plaintext secrets cannot be recovered. Reset the token if it is lost.',
    '请立即复制并保存。关闭窗口后无法再次查看。':'Copy and store it now. It cannot be viewed again after closing this window.',
    '为 AI Agent 或 CI 创建独立密钥。密钥仅显示一次。':'Create a dedicated credential for an AI agent or CI. The secret is shown once.',
    '平台管理员可管理所有空间；普通成员需要被授权到具体工作空间。':'Platform administrators manage all workspaces; members need explicit workspace access.',
    '角色决定成员在此工作空间中可以查看、发布或管理的内容。':'Roles determine what members can view, publish, or manage in this workspace.',
    '没有更多可添加用户。平台管理员可先在“用户与角色”中创建用户。':'There are no more users to add. A platform administrator can create one first.',
    '创建后，可在权限管理中将用户授权到工作空间。':'After creation, grant the user access from Access Management.',
    '创建成员，管理账号状态与平台级角色。':'Create members and manage account status and platform roles.',
    '按工作空间分配成员角色和访问范围。':'Assign member roles and access per workspace.',
    '选择用户及其在该工作空间中的角色。':'Select a user and their role in this workspace.',
    '平台管理员可管理所有工作空间':'Platform administrators can manage every workspace',
    '按住手柄上下拖动，松开后自动保存顺序':'Drag the handle to reorder; changes save automatically',
    '你可以完整管理自己创建的空间':'You can fully manage the workspaces you create',
    '空间标识创建后保持不变。':'The workspace slug remains unchanged after creation.',
    '工作空间名称':'Workspace name', '工作空间管理':'Workspace management', '我的工作空间':'My workspaces',
    '工作空间':'Workspaces', '我的空间':'My spaces', '空间设置':'Workspace settings', '平台管理':'Platform administration',
    '权限管理':'Access management', '用户管理':'User management', '平台角色':'Platform role', '空间角色':'Workspace role',
    '平台管理员':'Platform administrator', '平台成员':'Platform member', '普通成员':'Member',
    '所有者':'Owner', '空间管理员':'Workspace administrator', '编辑者':'Editor', '查看者':'Viewer',
    '完整管理权限':'Full management access', '成员、项目和 Token':'Members, projects, and tokens',
    '发布和管理页面':'Publish and manage pages', '只读访问内容':'Read-only content access',
    '创建平台管理员':'Create platform administrator', '创建并进入 Pages':'Create and enter Pages', '欢迎回来':'Welcome back',
    '你的姓名':'Your name', '姓名':'Name', '用户名':'Username', '密码':'Password', '邮箱':'Email', '选填':'Optional',
    '至少 10 个字符':'At least 10 characters', '登录':'Sign in', '退出登录':'Sign out', '退出':'Sign out',
    '语言':'Language', '界面语言':'Interface language', '切换语言':'Switch language',
    '外观':'Appearance', '界面主题':'Interface theme', '切换主题':'Switch theme', '浅色':'Light', '深色':'Dark', '自动':'Auto', '跟随系统':'Follow system',
    '当前深色，点击切换':'Dark mode; click to switch', '当前浅色，点击切换':'Light mode; click to switch',
    '服务正常':'Service healthy', '刷新':'Refresh', '正在载入…':'Loading…', '正在载入页面…':'Loading pages…',
    '正在载入工作空间…':'Loading workspaces…', '正在载入用户…':'Loading users…', '正在载入权限…':'Loading access…', '正在载入 Token…':'Loading tokens…',
    '还没有工作空间':'No workspaces yet', '还没有 API Token':'No API tokens yet', '还没有 API Token。':'No API tokens yet.', '暂无成员':'No members yet',
    '首页':'Home', '项目':'Project', '选择一个项目':'Select a project', '新建项目':'New project', '新建工作空间':'New workspace',
    '发布流程':'Publishing workflow', '创建项目':'Create project', '生成预览':'Generate preview', '正式发布':'Publish',
    '让 AI 发布页面':'Let AI publish pages', '空间项目':'Workspace projects', '页面':'Pages', '最近更新':'Last updated', '发布环境':'Environment',
    '创建预览版本':'Create preview version', '页面标识':'Page slug', '页面标题':'Page title', '产品发布页':'Product launch page',
    '生成预览':'Generate preview', '项目页面':'Project pages', '空间成员':'Workspace members', '最大 5MB':'5 MB maximum',
    '新版本':'New version', '版本历史':'Version history', '历史版本':'Previous versions', '线上版本':'Published version',
    '预览':'Preview', '发布':'Publish', '删除':'Delete', '编辑':'Edit', '查看':'View', '关闭':'Close', '取消':'Cancel', '确认':'Confirm',
    '保存':'Save', '已保存':'Saved', '移除':'Remove', '重置':'Reset', '吊销':'Revoke', '有效':'Active', '正常':'Active', '停用':'Disabled',
    '已停用':'Disabled', '已吊销':'Revoked', '已过期':'Expired', '尚未登录':'Never signed in', '从未使用':'Never used', '长期有效':'No expiration',
    '新增用户':'New user', '创建用户':'Create user', '初始密码':'Initial password', '重置密码':'Reset password', '留空则不修改':'Leave blank to keep unchanged',
    '用户':'User', '成员':'Member', '身份':'Role', '标识':'Slug', '项目':'Projects', '更新时间':'Updated', '状态':'Status', '最近登录':'Last sign-in',
    '加入时间':'Joined', '位平台用户':' platform users', '位已授权成员':' authorized members', '个工作空间':' workspaces',
    '授权用户':'Grant access', '确认授权':'Grant access', '添加用户':'Add user', '没有可授权的新用户，请先创建用户。':'No users are available. Create a user first.',
    '请先创建工作空间':'Create a workspace first', '请先选择工作空间':'Select a workspace first',
    '名称':'Name', 'Token 前缀':'Token prefix', '权限':'Permissions', '最后使用':'Last used', '创建时间':'Created', '有效期':'Expiration',
    '创建 Token':'Create token', 'Token 名称':'Token name', '读取 + 发布':'Read + publish', '只读':'Read only', '管理员':'Administrator',
    'Token 创建成功':'Token created', 'Token 已重置':'Token reset', '复制密钥':'Copy secret', '已复制':'Copied', '我已安全保存':'I stored it securely',
    '修改名称':'Rename', '保存名称':'Save name', '排序':'Order', '拖拽调整工作空间顺序':'Drag to reorder workspaces', '拖拽调整项目顺序':'Drag to reorder projects',
    '请求失败':'Request failed', '登录已过期，请重新登录':'Your session expired. Please sign in again.',
    '排序保存失败：':'Could not save order: ', '确认移除此用户的工作空间权限？':'Remove this user from the workspace?',
    '吊销后无法恢复，确认继续？':'Revocation cannot be undone. Continue?', '密钥仅显示一次。':'The secret is shown only once.',
    '创建':'Created', '个版本':' versions', '更新':'Updated', '已发布':'Published', '草稿':'Draft'
  };

  const replacements = Object.entries(dictionary).sort((a, b) => b[0].length - a[0].length);
  const translate = value => {
    if (language !== 'en' || !value) return value;
    let output = String(value);
    for (const [from, to] of replacements) output = output.split(from).join(to);
    return output
      .replace(/欢迎回到\s+(.+)/g, 'Welcome back to $1')
      .replace(/你以(.+)身份访问此空间。/g, 'You are accessing this workspace as $1. ')
      .replace(/管理\s+(.+)\s+的 AI Agent、CLI 和 CI\/CD 访问凭证。/g, 'Manage AI agent, CLI, and CI/CD credentials for $1.')
      .replace(/授权到\s+(.+)/g, 'Grant access to $1')
      .replace(/编辑\s+(.+)/g, 'Edit $1');
  };

  const skip = node => node.parentElement?.closest('#user-name,#project-name,#workspace-slug,#project-slug,.workspace>button,.projects>button,.home-project strong,[data-i18n-skip]');
  const localize = root => {
    if (language !== 'en') return;
    if (root.nodeType === Node.TEXT_NODE) {
      if (!skip(root)) root.nodeValue = translate(root.nodeValue);
      return;
    }
    if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_NODE) return;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let textNode;
    while ((textNode = walker.nextNode())) if (!skip(textNode)) textNode.nodeValue = translate(textNode.nodeValue);
    if (root.nodeType === Node.ELEMENT_NODE) {
      for (const attr of ['title', 'aria-label', 'placeholder']) if (root.hasAttribute(attr)) root.setAttribute(attr, translate(root.getAttribute(attr)));
      root.querySelectorAll?.('[title],[aria-label],[placeholder]').forEach(el => {
        for (const attr of ['title', 'aria-label', 'placeholder']) if (el.hasAttribute(attr)) el.setAttribute(attr, translate(el.getAttribute(attr)));
      });
    }
  };

  document.documentElement.lang = language;
  window.pagesLanguage = language;
  window.pagesLocale = language === 'en' ? 'en-US' : 'zh-CN';
  window.pagesTranslate = translate;
  const nativeAlert = window.alert.bind(window), nativeConfirm = window.confirm.bind(window), nativePrompt = window.prompt.bind(window);
  window.alert = message => nativeAlert(translate(message));
  window.confirm = message => nativeConfirm(translate(message));
  window.prompt = (message, value) => nativePrompt(translate(message), value);

  const choose = next => {
    localStorage.setItem(STORAGE_KEY, next === 'en' ? 'en' : 'zh-CN');
    location.reload();
  };
  document.querySelectorAll('[data-language-choice]').forEach(button => {
    button.classList.toggle('active', button.dataset.languageChoice === language);
    button.addEventListener('click', () => choose(button.dataset.languageChoice));
  });
  const authButton = document.querySelector('#auth-language');
  if (authButton) {
    authButton.textContent = language === 'en' ? '中文' : 'EN';
    authButton.addEventListener('click', () => choose(language === 'en' ? 'zh-CN' : 'en'));
  }
  localize(document);
  const observer = new MutationObserver(mutations => {
    observer.disconnect();
    for (const mutation of mutations) {
      if (mutation.type === 'characterData') localize(mutation.target);
      else mutation.addedNodes.forEach(localize);
    }
    observer.observe(document.body, {subtree:true, childList:true, characterData:true});
  });
  observer.observe(document.body, {subtree:true, childList:true, characterData:true});
})();
