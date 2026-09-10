package main

// dashboardHTML returns the self-contained resource page shown inside the
// management panel iframe. It reuses the panel management session from
// localStorage in memory only and never persists the management key.
func dashboardHTML() string {
	return dashboardPage
}

const dashboardPage = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Token Rhythm 账户</title>
<style>
:root{color-scheme:light dark;--bg:#f6f6f7;--fg:#1c1c1e;--muted:#6b7280;--card:#fff;--border:#e5e7eb;--accent:#2563eb;--good:#16a34a;--warn:#d97706;--bad:#dc2626}
@media (prefers-color-scheme:dark){:root{--bg:#101114;--fg:#e5e7eb;--muted:#9ca3af;--card:#17181c;--border:#2a2c33;--accent:#60a5fa}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI","Microsoft YaHei",sans-serif}
.wrap{max-width:1080px;margin:0 auto;padding:20px}
header{display:flex;justify-content:space-between;gap:12px;align-items:flex-end;flex-wrap:wrap}
h1{font-size:20px;margin:0}
h2{font-size:15px;margin:0 0 12px}
.sub{color:var(--muted);font-size:12px;margin-top:4px}
.actions{display:flex;gap:8px;flex-wrap:wrap;align-items:center}
input,button,select{font:inherit;padding:7px 10px;border-radius:8px;border:1px solid var(--border);background:var(--card);color:var(--fg)}
input,textarea{min-width:180px}
textarea{width:100%;min-height:64px;resize:vertical}
button{cursor:pointer}
button.primary{background:var(--accent);border-color:var(--accent);color:#fff}
button:disabled{opacity:.6;cursor:not-allowed}
.error{margin:14px 0;padding:10px 12px;border-radius:8px;border:1px solid var(--bad);background:rgba(220,38,38,.08);color:var(--bad)}
.okbox{margin:14px 0;padding:10px 12px;border-radius:8px;border:1px solid var(--good);background:rgba(22,163,74,.08)}
.okbox .secret{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;word-break:break-all;margin:8px 0;font-size:13px}
.sessions{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px;margin:16px 0}
.session{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:12px;text-align:left;cursor:pointer;width:100%}
.session.active{border-color:var(--accent);box-shadow:0 0 0 1px var(--accent)}
.session .name{font-weight:600}
.session .meta{color:var(--muted);font-size:12px;margin-top:4px}
.session .bal{font-size:18px;font-weight:600;margin-top:6px}
.session.low .bal{color:var(--bad)}
.session.fail .bal{color:var(--bad);font-size:13px;font-weight:500}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:12px;margin:18px 0}
.card{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:14px}
.card .label{color:var(--muted);font-size:12px}
.card .value{font-size:22px;font-weight:600;margin-top:6px;word-break:break-all}
.card.low .value{color:var(--bad)}
.card.good .value{color:var(--good)}
.panel{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:16px;margin-bottom:16px}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:12px}
.kv .k{color:var(--muted);font-size:12px}
.kv .v{font-size:16px;font-weight:600;margin-top:2px;word-break:break-all}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{text-align:left;padding:7px 8px;border-bottom:1px solid var(--border);white-space:nowrap}
th{color:var(--muted);font-weight:500}
td.num,th.num{text-align:right}
.table-wrap{overflow-x:auto}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-bottom:12px}
footer{color:var(--muted);font-size:12px;text-align:center;padding:8px 0 24px}
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div>
      <h1>Token Rhythm 账户</h1>
      <div class="sub" id="sub">正在加载…</div>
    </div>
    <div class="actions">
      <input id="key" type="password" placeholder="管理密钥（可留空自动复用面板会话）">
      <button id="saveKey">保存密钥</button>
      <button id="refresh" class="primary">刷新全部</button>
    </div>
  </header>
  <div id="error" class="error" hidden></div>
  <div id="created" class="okbox" hidden></div>
  <section class="panel">
    <h2>添加 Session</h2>
    <p class="sub">在 Token Rhythm 浏览器里复制 Cookie，粘贴到这里保存。写入的是 CPA 插件配置，不用改 yaml。创建 API Key 需要包含 <code>tr_csrf</code>。</p>
    <div class="row">
      <input id="sessName" placeholder="名称（可选，如 主账号）">
      <button id="saveSess" class="primary">保存 Session</button>
    </div>
    <textarea id="sessCookie" placeholder="tr_session=sess_xxx; tr_csrf=yyy&#10;或只贴 sess_xxx（仅能查余额，不能创建 Key）"></textarea>
  </section>
  <section class="sessions" id="sessionCards"></section>
  <section class="cards" id="cards"></section>
  <section class="panel">
    <h2>账户概览</h2>
    <div class="grid" id="overview"></div>
  </section>
  <section class="panel">
    <h2>用量统计</h2>
    <div class="grid" id="usage"></div>
  </section>
  <section class="panel">
    <h2>API Keys</h2>
    <div class="row">
      <input id="keyName" placeholder="新 Key 名称（可留空自动命名）">
      <button id="createKey" class="primary">创建 API Key</button>
    </div>
    <div class="table-wrap" id="keys"></div>
  </section>
  <section class="panel" id="modelsPanel" hidden>
    <h2>按模型用量</h2>
    <div class="table-wrap" id="models"></div>
  </section>
  <section class="panel" id="rewardPanel" hidden>
    <h2>新人奖励</h2>
    <div class="grid" id="reward"></div>
  </section>
  <section class="panel" id="expiringPanel" hidden>
    <h2>即将过期明细</h2>
    <div class="table-wrap" id="expiring"></div>
  </section>
  <footer id="foot"></footer>
</div>
<script>
(function(){
  'use strict';
  var PREFIX='enc::v1::';
  var SALT='cli-proxy-api-webui::secure-storage';
  var KEY_STORAGE='tokenrhythm-balance.managementKey';
  var SELECTED_STORAGE='tokenrhythm-balance.selectedSession';
  var timer=null;
  var selectedId=localStorage.getItem(SELECTED_STORAGE)||'';
  var latest=null;
  var creating=false;

  function keyBytes(){
    return new TextEncoder().encode(SALT+'|'+location.host+'|'+navigator.userAgent);
  }
  function deobfuscate(payload){
    if(!payload || payload.indexOf(PREFIX)!==0){return payload;}
    try{
      var body=atob(payload.slice(PREFIX.length));
      var bytes=new Uint8Array(body.length);
      for(var i=0;i<body.length;i++){bytes[i]=body.charCodeAt(i);}
      var kb=keyBytes();
      for(var j=0;j<bytes.length;j++){bytes[j]^=kb[j%kb.length];}
      return new TextDecoder().decode(bytes);
    }catch(e){return payload;}
  }
  function panelAuth(){
    try{
      var raw=localStorage.getItem('cli-proxy-auth');
      if(!raw){return null;}
      var data=JSON.parse(deobfuscate(raw));
      var state=(data&&data.state)?data.state:data;
      if(!state){return null;}
      return {apiBase:state.apiBase||'',managementKey:state.managementKey||''};
    }catch(e){return null;}
  }
  function managementKey(){
    var auth=panelAuth();
    if(auth&&auth.managementKey){return auth.managementKey;}
    return localStorage.getItem(KEY_STORAGE)||'';
  }
  function apiBase(){
    var auth=panelAuth();
    if(auth&&auth.apiBase){return auth.apiBase.replace(/\/+$/,'');}
    if(/^manager\./i.test(location.hostname)){
      return location.origin.replace(/\/\/manager\./i,'//cpa.');
    }
    return '';
  }
  function authHeaders(){
    var key=managementKey();
    return {'Authorization':'Bearer '+key,'X-Management-Key':key};
  }
  function balanceURL(force){
    var q=[];
    if(force){q.push('refresh=1');}
    if(selectedId){q.push('session='+encodeURIComponent(selectedId));}
    return apiBase()+'/v0/management/tokenrhythm/balance'+(q.length?'?'+q.join('&'):'');
  }
  function createURL(){
    return apiBase()+'/v0/management/tokenrhythm/api-keys';
  }
  function pluginConfigURL(){
    return apiBase()+'/v0/management/plugins/tokenrhythm-balance/config';
  }
  function el(id){return document.getElementById(id);}
  function esc(v){return String(v==null?'':v).replace(/[&<>"']/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];});}
  function money(v){
    if(v==null||v===''){return '-';}
    var n=parseFloat(v);
    if(isNaN(n)){return String(v);}
    return n.toFixed(2);
  }
  function int(v){
    var n=parseInt(v,10);
    if(isNaN(n)){return '0';}
    return n.toLocaleString('zh-CN');
  }
  function card(label,value,cls){
    return '<div class="card '+(cls||'')+'"><div class="label">'+esc(label)+'</div><div class="value">'+value+'</div></div>';
  }
  function kv(k,v){
    return '<div class="kv"><div class="k">'+esc(k)+'</div><div class="v">'+v+'</div></div>';
  }
  function table(headers,rows){
    var head='<tr>'+headers.map(function(x,i){return '<th'+(i>0?' class="num"':'')+'>'+esc(x)+'</th>';}).join('')+'</tr>';
    var body=rows.map(function(r){return '<tr>'+r.map(function(c,i){return '<td'+(i>0?' class="num"':'')+'>'+c+'</td>';}).join('')+'</tr>';}).join('');
    return '<table>'+head+body+'</table>';
  }
  function showError(msg){
    var box=el('error');
    if(!msg){box.hidden=true;box.textContent='';return;}
    box.hidden=false;box.textContent=msg;
  }
  function sessionList(data){
    if(data&&data.sessions&&data.sessions.length){return data.sessions;}
    return [{
      id:data.selected_session||'default',
      name:(data.account&&data.account.name)||'default',
      ok:!!data.ok,
      error:data.error||'',
      available:data.available||0,
      low_balance:!!data.low_balance,
      account_name:data.account&&data.account.name||'',
      api_key_count:(data.api_keys||[]).length,
      has_csrf:true,
      wallet:data.wallet,
      usage:data.usage,
      panel:data.panel,
      account:data.account,
      expiring:data.expiring,
      api_keys:data.api_keys||[]
    }];
  }
  function pickSession(data){
    var list=sessionList(data);
    var found=null;
    for(var i=0;i<list.length;i++){
      if(list[i].id===selectedId){found=list[i];break;}
    }
    if(!found){found=list[0];}
    if(found){selectedId=found.id;localStorage.setItem(SELECTED_STORAGE,selectedId);}
    return found||{};
  }
  function renderSessions(data){
    var list=sessionList(data);
    el('sessionCards').innerHTML=list.map(function(s){
      var cls='session'+(s.id===selectedId?' active':'')+(s.ok&&s.low_balance?' low':'')+(s.ok?'':' fail');
      var bal=s.ok?esc(money(s.available)):(esc(s.error||'查询失败'));
      var meta=(s.account_name?esc(s.account_name)+' · ':'')+'Key '+(s.api_key_count||0)+(s.has_csrf?'':' · 缺 CSRF');
      return '<button type="button" class="'+cls+'" data-id="'+esc(s.id)+'"><div class="name">'+esc(s.name||s.id)+'</div><div class="bal">'+bal+'</div><div class="meta">'+meta+'</div></button>';
    }).join('');
    var buttons=el('sessionCards').querySelectorAll('button[data-id]');
    for(var i=0;i<buttons.length;i++){
      buttons[i].addEventListener('click',function(){
        selectedId=this.getAttribute('data-id')||'';
        localStorage.setItem(SELECTED_STORAGE,selectedId);
        if(latest){render(latest);}
      });
    }
  }
  function renderKeys(sess){
    var keys=sess.api_keys||[];
    if(!keys.length){
      el('keys').innerHTML='<div class="sub">该账号暂无 API Key。</div>';
      return;
    }
    var rows=keys.map(function(k){
      return [esc(k.name||'-'),esc(k.maskedKey||k.keyPrefix||'-'),esc(k.status||'-'),esc(k.createdAt||'-'),esc(k.lastUsedAt||'-')];
    });
    el('keys').innerHTML=table(['名称','Key','状态','创建时间','最近使用'],rows);
  }
  function render(data){
    latest=data;
    var sess=pickSession(data);
    renderSessions(data);
    var wallet=sess.wallet||data.wallet||{};
    var usage=sess.usage||data.usage||{};
    var panel=sess.panel||data.panel||{};
    var ps=panel.summary||{};
    var account=sess.account||data.account||{};
    var expiring=sess.expiring||data.expiring||{};
    var expSummary=expiring.summary||{};
    var currency=wallet.currency||usage.currency||'CNY';
    renderKeys(sess);
    el('cards').innerHTML=
      card('可用余额 ('+esc(currency)+')',esc(money(wallet.availableBalanceCny)),sess.low_balance?'low':'good')+
      card('赠送余额',esc(money(wallet.giftAvailableCny)))+
      card('充值余额',esc(money(wallet.rechargeBalanceCny)))+
      card('冻结余额',esc(money(wallet.frozenBalanceCny)));
    var accountLine=account.name?esc(account.name)+(account.phoneMasked?' · '+esc(account.phoneMasked):''):'';
    el('overview').innerHTML=
      (accountLine?kv('账号信息',accountLine):'')+
      kv('账户状态',esc(account.status||wallet.giftStatus||'-'))+
      kv('累计消耗 ('+esc(currency)+')',esc(money(usage.costCny||ps.costCny)))+
      kv('下次过期时间',esc(usage.nextExpiryAt||expSummary.nextExpiryAt||'-'))+
      kv('即将过期额度',esc(money(usage.expiringBalanceCny||expSummary.expiringBalanceCny)))+
      kv('数据时间',esc(wallet.asOf||data.fetched_at||'-'));
    var calls=usage.calls||ps.calls||0;
    var success=usage.successCalls||ps.successCalls||0;
    var errors=usage.errorCalls||ps.errorCalls||0;
    var inputTokens=usage.inputTokens||ps.inputTokens||0;
    var outputTokens=usage.outputTokens||ps.outputTokens||0;
    var cacheRead=ps.cacheReadTokens||0;
    var rate=calls>0?((success/calls)*100).toFixed(1)+'%':'-';
    el('usage').innerHTML=
      kv('总调用次数',esc(int(calls)))+
      kv('成功 / 失败',esc(int(success))+' / '+esc(int(errors)))+
      kv('成功率',esc(rate))+
      kv('输入 Tokens',esc(int(inputTokens)))+
      kv('输出 Tokens',esc(int(outputTokens)))+
      kv('缓存读取 Tokens',esc(int(cacheRead)))+
      kv('合计 Tokens',esc(int(ps.totalTokens||(inputTokens+outputTokens))))+
      kv('折合 USD',ps.actualCostUsd?('$'+esc(money(ps.actualCostUsd))):'-');
    var models=panel.byModel||[];
    if(models.length){
      el('modelsPanel').hidden=false;
      var mrows=models.slice(0,30).map(function(m){
        return [esc(m.model||m.modelId||'-'),esc(int(m.calls)),esc(int(m.inputTokens)),esc(int(m.outputTokens)),esc(int(m.cacheReadTokens||0)),esc(money(m.costCny))];
      });
      el('models').innerHTML=table(['模型','次数','输入','输出','缓存读','费用('+currency+')'],mrows);
    }else{
      el('modelsPanel').hidden=true;
    }
    var credits=expiring.list||[];
    if(credits.length){
      el('expiringPanel').hidden=false;
      var erows=credits.map(function(c){
        return [esc(c.sourceLabel||c.source||'-'),esc(money(c.grantedCny)),esc(money(c.remainingCny)),esc(c.expiresAt||'-')];
      });
      el('expiring').innerHTML=table(['来源','发放','剩余','过期时间'],erows);
    }else{
      el('expiringPanel').hidden=true;
    }
    var reward=usage.signupReward;
    if(reward){
      el('rewardPanel').hidden=false;
      el('reward').innerHTML=
        kv('状态',esc(reward.status||'-'))+
        kv('已发放',esc(money(reward.grantedCny)))+
        kv('总可享',esc(money(reward.totalEligibleCny)))+
        kv('有效期(天)',esc(reward.rewardValidityDays||'-'))+
        kv('发放时间',esc(reward.activationRewardGrantedAt||'-'))+
        kv('过期时间',esc(reward.activationRewardExpiresAt||'-'));
    }else{
      el('rewardPanel').hidden=true;
    }
    var interval=data.refresh_interval_seconds||60;
    var who=account.name?('账号 '+account.name+' · '):'';
    var count=data.session_count||sessionList(data).length;
    el('sub').textContent=who+'共 '+count+' 个 session · 每 '+interval+' 秒轮询 · 并发 '+(data.poll_concurrency||3)+' · 数据时间 '+(data.fetched_at||'');
    el('foot').textContent='数据来源：'+(data.base_url||'')+' · 阈值 '+(data.low_balance_threshold||0)+' · 创建 Key 需要 cookie 中包含 tr_csrf';
    schedule(interval);
  }
  function schedule(interval){
    if(timer){clearInterval(timer);}
    timer=setInterval(function(){load(false);},Math.max(5,interval)*1000);
  }
  function load(force){
    var key=managementKey();
    if(!key){
      showError('未找到管理密钥：请在面板登录以复用会话，或在上方输入管理密钥后点击“保存密钥”。');
      return;
    }
    el('sub').textContent='正在轮询 session…';
    fetch(balanceURL(force),{headers:authHeaders(),credentials:'same-origin'})
      .then(parseResp)
      .then(function(data){
        if(data&&(data.ok||(data.sessions&&data.sessions.length))){showError(data.ok?'':(data.error||''));render(data);}
        else{showError((data&&data.error)||'查询失败。若刚保存过 Session，等一两秒再刷新；并确认 Linux 上的 CPA 能访问 tokenrhythm.studio。');el('sub').textContent='刷新失败';}
      })
      .catch(function(err){showError('请求失败：'+err);el('sub').textContent='刷新失败';});
  }
  function showCreated(key){
    var box=el('created');
    if(!key||!key.key){box.hidden=true;box.innerHTML='';return;}
    box.hidden=false;
    box.innerHTML='<strong>API Key 已创建：'+esc(key.name||'')+'</strong><div class="secret" id="secret">'+esc(key.key)+'</div><div class="sub">完整 Key 只展示一次，请立即复制保存。</div><button type="button" id="copySecret">复制 Key</button>';
    var btn=el('copySecret');
    if(btn){
      btn.addEventListener('click',function(){
        var text=key.key;
        if(navigator.clipboard&&navigator.clipboard.writeText){
          navigator.clipboard.writeText(text);
        }
      });
    }
  }
  el('saveKey').addEventListener('click',function(){
    var value=el('key').value.trim();
    if(value){localStorage.setItem(KEY_STORAGE,value);}
    else{localStorage.removeItem(KEY_STORAGE);}
    load(true);
  });
  el('refresh').addEventListener('click',function(){load(true);});
  function parseResp(resp){
    return resp.text().then(function(text){
      var data=null;
      try{data=JSON.parse(text);}catch(e){
        var plain=String(text||'').replace(/<[^>]+>/g,' ').replace(/\s+/g,' ').trim();
        return {ok:false,error:'HTTP '+resp.status+(plain?(' '+plain.slice(0,240)):'')};
      }
      if(data && !data.ok && !data.error){
        data.error=data.message||('HTTP '+resp.status);
      }
      return data||{ok:false,error:'HTTP '+resp.status};
    });
  }
  function jsonHeaders(){
    return Object.assign({'Content-Type':'application/json','Accept':'application/json'},authHeaders());
  }
  function sessionSlug(name){
    var s=String(name||'').toLowerCase().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'');
    return s||'';
  }
  function saveSessionToCPA(){
    var cookie=el('sessCookie').value.trim();
    var name=el('sessName').value.trim();
    if(!cookie){showError('请粘贴 Token Rhythm 的 session Cookie。');return;}
    if(!managementKey()){showError('未找到管理密钥。');return;}
    el('saveSess').disabled=true;
    fetch(pluginConfigURL(),{headers:authHeaders(),credentials:'same-origin'})
      .then(function(resp){return resp.json().catch(function(){return {};});})
      .then(function(cfg){
        if(!cfg||typeof cfg!=='object'||cfg.error){cfg={};}
        var sessions=Array.isArray(cfg.sessions)?cfg.sessions.slice():[];
        if(!sessions.length && cfg.tr_session){
          sessions.push({id:'default',name:'default',tr_session:String(cfg.tr_session)});
        }
        var id=sessionSlug(name);
        if(!id){id=sessions.length?'session-'+(sessions.length+1):'default';}
        var found=-1;
        for(var i=0;i<sessions.length;i++){
          if(sessions[i]&&(sessions[i].id===id||sessions[i].tr_session===cookie)){found=i;break;}
        }
        var item={id:id,name:name||id,tr_session:cookie};
        if(found>=0){sessions[found]=item;}
        else{sessions.push(item);}
        return fetch(pluginConfigURL(),{
          method:'PATCH',
          headers:jsonHeaders(),
          credentials:'same-origin',
          body:JSON.stringify({enabled:true,sessions:sessions,tr_session:null})
        });
      })
      .then(function(resp){
        if(!resp){return;}
        return parseResp(resp).then(function(data){
          if(resp.ok && (!data||!data.error)){
            showError('');
            el('sessCookie').value='';
            el('created').hidden=false;
            el('created').innerHTML='<strong>Session 已保存到 CPA 插件配置。</strong><div class="sub">正在重新轮询…</div>';
            setTimeout(function(){load(true);},1200);
            return;
          }
          showError((data&&(data.message||data.error))||('保存失败 HTTP '+resp.status));
        });
      })
      .catch(function(err){showError('保存 Session 失败：'+err);})
      .then(function(){el('saveSess').disabled=false;});
  }
  el('saveSess').addEventListener('click',saveSessionToCPA);
  el('createKey').addEventListener('click',function(){
    if(creating){return;}
    var key=managementKey();
    if(!key){showError('未找到管理密钥。');return;}
    if(!selectedId){showError('请先选择一个 session。');return;}
    creating=true;
    el('createKey').disabled=true;
    fetch(createURL(),{
      method:'POST',
      headers:Object.assign({'Content-Type':'application/json'},authHeaders()),
      credentials:'same-origin',
      body:JSON.stringify({session:selectedId,name:el('keyName').value.trim()})
    })
      .then(parseResp)
      .then(function(data){
        if(data&&data.ok&&data.api_key){
          showError('');
          showCreated(data.api_key);
          el('keyName').value='';
          load(true);
        }else{
          showError((data&&data.error)||'创建 API Key 失败');
        }
      })
      .catch(function(err){showError('创建失败：'+err);})
      .then(function(){creating=false;el('createKey').disabled=false;});
  });
  var auth=panelAuth();
  if(auth&&auth.managementKey){el('key').placeholder='已复用面板管理会话';}
  load(false);
})();
</script>
</body>
</html>
`
