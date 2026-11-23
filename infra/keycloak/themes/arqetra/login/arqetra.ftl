<#macro intro active>
  <#if locale?? && realm.internationalizationEnabled && locale.supported?size gt 1>
    <nav class="arqetra-locale" aria-label="${msg("arqetraLocaleLabel")}">
      <#list ["uk", "en"] as tag>
        <#list locale.supported as option>
          <#if option.languageTag == tag>
            <a href="${option.url}" lang="${tag}" aria-label="${option.label}"<#if tag == locale.currentLanguageTag> aria-current="true"</#if>>${msg("arqetraLocale_" + tag)}</a>
          </#if>
        </#list>
      </#list>
    </nav>
  </#if>
  <#if realm.password && realm.registrationAllowed && !registrationDisabled??>
    <nav class="arqetra-tabs" aria-label="${msg("arqetraTabsLabel")}">
      <a href="${url.loginUrl}"<#if active == "login"> aria-current="page"</#if>>${msg("arqetraTabLogin")}</a>
      <a href="${url.registrationUrl}"<#if active == "register"> aria-current="page"</#if>>${msg("arqetraTabRegister")}</a>
    </nav>
  </#if>
</#macro>
