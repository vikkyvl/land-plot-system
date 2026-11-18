<#macro intro active>
  <p class="arqetra-intro">${msg("arqetraIntro")}</p>
  <#if realm.password && realm.registrationAllowed && !registrationDisabled??>
    <nav class="arqetra-tabs" aria-label="${msg("arqetraTabsLabel")}">
      <a href="${url.loginUrl}"<#if active == "login"> aria-current="page"</#if>>${msg("arqetraTabLogin")}</a>
      <a href="${url.registrationUrl}"<#if active == "register"> aria-current="page"</#if>>${msg("arqetraTabRegister")}</a>
    </nav>
  </#if>
</#macro>
