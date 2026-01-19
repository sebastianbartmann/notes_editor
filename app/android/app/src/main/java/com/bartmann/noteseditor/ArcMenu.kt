package com.bartmann.noteseditor

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.scale
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Menu
import kotlin.math.cos
import kotlin.math.roundToInt
import kotlin.math.sin

private val ARC_RADIUS_LEVEL1 = 90.dp
private val ARC_RADIUS_LEVEL2 = 180.dp
private val MENU_BUTTON_SIZE = 56.dp
private val MENU_ITEM_SIZE = 48.dp
private val ICON_SIZE = 24.dp
private const val START_ANGLE = 180f
private const val SWEEP_ANGLE = 90f
private const val ANIMATION_DURATION = 150
private const val LEVEL_TRANSITION_DURATION = 200

@Composable
fun ArcMenuButton(
    isExpanded: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val interactionSource = remember { MutableInteractionSource() }
    val isPressed by interactionSource.collectIsPressedAsState()
    val scale = if (isPressed) 0.95f else 1f
    val icon = if (isExpanded) Icons.Default.Close else Icons.Default.Menu
    val description = if (isExpanded) "Close menu" else "Open navigation menu"

    Box(
        modifier = modifier
            .size(MENU_BUTTON_SIZE)
            .scale(scale)
            .shadow(6.dp, CircleShape)
            .background(AppTheme.colors.accent, CircleShape)
            .clickable(
                interactionSource = interactionSource,
                indication = null
            ) { onClick() }
            .semantics { contentDescription = description },
        contentAlignment = Alignment.Center
    ) {
        Image(
            painter = rememberVectorPainter(icon),
            contentDescription = null,
            colorFilter = ColorFilter.tint(AppTheme.colors.buttonText),
            modifier = Modifier.size(ICON_SIZE)
        )
    }
}

@Composable
fun ArcMenuItemView(
    item: ArcMenuItem,
    isActive: Boolean,
    isMoreItem: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val interactionSource = remember { MutableInteractionSource() }
    val isPressed by interactionSource.collectIsPressedAsState()
    val scale = if (isPressed) 0.95f else 1f
    val iconColor = if (isActive) AppTheme.colors.accent else AppTheme.colors.muted
    val bgColor = if (isMoreItem) AppTheme.colors.accentDim else AppTheme.colors.panel

    Column(
        modifier = modifier
            .sizeIn(minWidth = MENU_ITEM_SIZE, minHeight = MENU_ITEM_SIZE)
            .scale(scale)
            .shadow(4.dp, RoundedCornerShape(8.dp))
            .background(bgColor, RoundedCornerShape(8.dp))
            .clickable(
                interactionSource = interactionSource,
                indication = null
            ) { onClick() }
            .padding(horizontal = 8.dp, vertical = 6.dp)
            .semantics { contentDescription = "${item.label} navigation" },
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Image(
            painter = rememberVectorPainter(item.icon),
            contentDescription = null,
            colorFilter = ColorFilter.tint(iconColor),
            modifier = Modifier.size(ICON_SIZE)
        )
        AppText(
            text = item.label,
            style = AppTheme.typography.label,
            color = AppTheme.colors.text
        )
    }
}

/**
 * Calculate the position of an arc menu item using polar coordinates.
 *
 * @param index Item index (0 = leftmost position)
 * @param itemCount Total items in current level
 * @param radiusPx Distance from corner to item center in pixels
 * @param startAngle Arc start angle in degrees (180 = left)
 * @param sweepAngle Total arc sweep in degrees (90 = quarter circle)
 * @return IntOffset for the item position relative to bottom-right corner
 */
private fun calculateItemPosition(
    index: Int,
    itemCount: Int,
    radiusPx: Float,
    startAngle: Float = START_ANGLE,
    sweepAngle: Float = SWEEP_ANGLE
): IntOffset {
    val angleStep = if (itemCount > 1) sweepAngle / (itemCount - 1) else 0f
    val angle = startAngle - (index * angleStep)
    val angleRadians = Math.toRadians(angle.toDouble())

    // Negate Y because screen coordinates have Y increasing downward
    return IntOffset(
        x = (radiusPx * cos(angleRadians)).roundToInt(),
        y = -(radiusPx * sin(angleRadians)).roundToInt()
    )
}

@Composable
fun ArcMenu(
    items: List<ArcMenuItem>,
    currentRoute: String?,
    menuState: ArcMenuState,
    onStateChange: (ArcMenuState) -> Unit,
    onNavigate: (String) -> Unit,
    onOpenExternal: (String) -> Unit,
    modifier: Modifier = Modifier,
    showButton: Boolean = true
) {
    val density = LocalDensity.current
    val expandProgress = remember { Animatable(0f) }

    LaunchedEffect(menuState) {
        val targetValue = when (menuState) {
            ArcMenuState.COLLAPSED -> 0f
            ArcMenuState.LEVEL1 -> 1f
            ArcMenuState.LEVEL2 -> 1f
        }
        expandProgress.animateTo(
            targetValue = targetValue,
            animationSpec = tween(
                durationMillis = ANIMATION_DURATION,
                easing = FastOutSlowInEasing
            )
        )
    }

    val moreItem = items.find { it.id == "more" }
    val level1Items = items
    val level2Items = moreItem?.children ?: emptyList()
    val showLevel1 = menuState == ArcMenuState.LEVEL1 || menuState == ArcMenuState.LEVEL2
    val showLevel2 = menuState == ArcMenuState.LEVEL2

    fun handleItemTap(item: ArcMenuItem) {
        when {
            item.id == "back" -> {
                onStateChange(ArcMenuState.LEVEL1)
            }
            item.children != null -> {
                onStateChange(ArcMenuState.LEVEL2)
            }
            item.externalUrl != null -> {
                onOpenExternal(item.externalUrl)
                onStateChange(ArcMenuState.COLLAPSED)
            }
            item.route != null -> {
                onNavigate(item.route)
                onStateChange(ArcMenuState.COLLAPSED)
            }
        }
    }

    Box(modifier = modifier.fillMaxSize()) {
        // Scrim overlay when expanded
        if (menuState != ArcMenuState.COLLAPSED) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = 0.2f * expandProgress.value))
                    .pointerInput(Unit) {
                        detectTapGestures {
                            onStateChange(ArcMenuState.COLLAPSED)
                        }
                    }
            )
        }

        // Menu positioned to align with BottomInfoBar button
        Box(
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(start = 16.dp, end = 80.dp, top = 16.dp, bottom = 16.dp)
        ) {
            val level1RadiusPx = with(density) { ARC_RADIUS_LEVEL1.toPx() }
            val level2RadiusPx = with(density) { ARC_RADIUS_LEVEL2.toPx() }

            // Level 1 items (inner ring)
            if (showLevel1) {
                level1Items.forEachIndexed { index, item ->
                    val position = calculateItemPosition(
                        index = index,
                        itemCount = level1Items.size,
                        radiusPx = level1RadiusPx * expandProgress.value
                    )
                    val isActive = item.route == currentRoute
                    val isMoreItem = item.id == "more"

                    Box(
                        modifier = Modifier
                            .offset { position }
                            .alpha(expandProgress.value)
                    ) {
                        ArcMenuItemView(
                            item = item,
                            isActive = isActive,
                            isMoreItem = isMoreItem,
                            onClick = { handleItemTap(item) }
                        )
                    }
                }
            }

            // Level 2 items (outer ring)
            if (showLevel2) {
                level2Items.forEachIndexed { index, item ->
                    val position = calculateItemPosition(
                        index = index,
                        itemCount = level2Items.size,
                        radiusPx = level2RadiusPx * expandProgress.value
                    )
                    val isActive = item.route == currentRoute

                    Box(
                        modifier = Modifier
                            .offset { position }
                            .alpha(expandProgress.value)
                    ) {
                        ArcMenuItemView(
                            item = item,
                            isActive = isActive,
                            isMoreItem = false,
                            onClick = { handleItemTap(item) }
                        )
                    }
                }
            }

            // Menu button (conditionally rendered)
            if (showButton) {
                ArcMenuButton(
                    isExpanded = menuState != ArcMenuState.COLLAPSED,
                    onClick = {
                        when (menuState) {
                            ArcMenuState.COLLAPSED -> onStateChange(ArcMenuState.LEVEL1)
                            else -> onStateChange(ArcMenuState.COLLAPSED)
                        }
                    }
                )
            }
        }
    }
}
